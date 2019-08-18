// Copyright 2019 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package maistra

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup24(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.ShellMuteOutput("oc delete rule/keyval -n " + meshNamespace)
	log.Info("Waiting for rules to be cleaned up. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
	util.ShellMuteOutput("oc delete handler/keyval instance/keyval adapter/keyval template/keyval -n " + meshNamespace)
	util.ShellMuteOutput("oc delete service keyval -n " + meshNamespace)
	util.ShellMuteOutput("oc delete deployment keyval -n " + meshNamespace)
	util.ShellMuteOutput("oc delete dc keyval -n " + meshNamespace)
	util.ShellSilent("rm -f /tmp/mesh.yaml")

	util.KubeDelete(namespace, httpbinPolicyAllYaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}


func checkUserGroup(url, ingress, ingressPort, user string) (*http.Response, error) {
	// Declare http client
	client := &http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set header key user
	req.Header.Set("user", user)
	// Get response
	return client.Do(req)
}


func Test24(t *testing.T) {
	defer cleanup24(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_24 Control Headers and Routing")
	util.Inspect(util.KubeApply(testNamespace, httpbinPolicyAllYaml, kubeconfigFile), "failed to deploy httpbin", "", t)
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	util.CheckPodRunning(testNamespace, "app=httpbin", kubeconfigFile)

	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", meshNamespace, kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	ingressPort, err := util.GetIngressPort(meshNamespace, "istio-ingressgateway", kubeconfigFile)
	util.Inspect(err, "cannot get ingress port", "", t)

	log.Info("Enable policy check")
	util.ShellMuteOutput("oc -n " + meshNamespace + " get cm istio -o jsonpath=\"{@.data.mesh}\" | sed -e \"s@disablePolicyChecks: true@disablePolicyChecks: false@\" > /tmp/mesh.yaml")
	util.ShellMuteOutput("oc -n " + meshNamespace + " create cm istio -o yaml --dry-run --from-file=mesh=/tmp/mesh.yaml | oc replace -f -")
	log.Info("Verify disablePolicyChecks should be false")
	util.Shell("oc -n " + meshNamespace + " get cm istio -o jsonpath=\"{@.data.mesh}\" | grep disablePolicyChecks")

	log.Info("Output Producing Adapters")

	util.Shell("oc run keyval --image=gcr.io/istio-testing/keyval:release-1.1 --namespace " + meshNamespace + " --port 9070 --expose")
	util.Inspect(util.KubeApply(meshNamespace, httpbinKeyvalTemplateYaml, kubeconfigFile), "failed to apply keyval template", "",t)
	util.Inspect(util.KubeApply(meshNamespace, httpbinKeyvalYaml, kubeconfigFile), "failed to apply keyval", "", t)

	log.Info("Create a rule for adapter")
	util.Inspect(util.KubeApplyContents(meshNamespace, demoAdapter, kubeconfigFile), "failed to apply adapter handler", "", t)
	util.Inspect(util.KubeApplyContents(meshNamespace, keyvalInstance, kubeconfigFile), "failed to apply keyval instance", "", t)
		
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	t.Run("access_httpbin_headers_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Check httpbin headers")
		resp, duration, err := util.GetHTTPResponse(fmt.Sprintf("http://%s:%s/headers", ingress, ingressPort), nil)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("httpbin headers page returned in %d ms", duration)
		util.Inspect(util.CheckHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		log.Infof("%v", string(body))
	})

	t.Run("httpbin_user_group_headers_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()
		
		log.Info("Create a demo adapter rule")
		util.Inspect(util.KubeApplyContents(meshNamespace, demoAdapterRule, kubeconfigFile), "failed to apply demo adapter rule", "", t)
		log.Info("Waiting for rules to propagate. Sleep 70 seconds...")
		time.Sleep(time.Duration(70) * time.Second)

		log.Info("Verify that user:json has \"User-Group\": \"admin\" header")
		resp, err := checkUserGroup(fmt.Sprintf("http://%s:%s/headers", ingress, ingressPort), ingress, ingressPort, "jason")
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		if strings.Contains(string(body), "\"User-Group\": \"admin\"") {
			log.Infof("Get expected headers: %s", string(body))
		} else {
			t.Errorf("Missing User-Group: admin headers: %s", string(body))
			log.Errorf("Missing User-Group: admin headers: %s", string(body))
		}
	})

	t.Run("httpbin_418_headers_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Redirect the URI path to a 418 virtual service")
		util.Inspect(util.KubeApplyContents(meshNamespace, demoAdapterRule2, kubeconfigFile), "failed to apply demo adapter rule 2", "", t)
		log.Info("Waiting for rules to propagate. Sleep 40 seconds...")
		time.Sleep(time.Duration(40) * time.Second)

		log.Info("Verify response 418 teapot")
		resp, err := checkUserGroup(fmt.Sprintf("http://%s:%s/headers", ingress, ingressPort), ingress, ingressPort, "jason")
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)
		if strings.Contains(string(body), "teapot") {
			log.Infof("Get expected headers: %s", string(body))
		} else {
			t.Errorf("Expect teapot response. Got unexpected response: %s", string(body))
			log.Errorf("Expect teapot response. Got unexpected response: %s", string(body))
		}
	})
	
}
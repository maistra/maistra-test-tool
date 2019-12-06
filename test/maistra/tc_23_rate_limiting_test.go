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
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup23(namespace string, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDeleteContents(meshNamespace, rateLimitYaml, kubeconfig)
	time.Sleep(time.Duration(30) * time.Second)
	util.KubeDelete(namespace, bookinfoAllv1Yaml, kubeconfig)
	util.ShellSilent("rm -f /tmp/mesh.yaml")
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}



func Test23(t *testing.T) {
	defer cleanup23(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()
	
	log.Infof("# TC_23 Rate Limitting")
	updateYaml()

	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", meshNamespace, kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)
	productpageURL := fmt.Sprintf("http://%s/productpage", ingress)
	testUserJar := util.GetCookieJar(testUsername, "", "http://"+ingress)
	
	log.Info("Enable policy check")
	util.ShellMuteOutput("oc -n " + meshNamespace + " get cm istio -o jsonpath=\"{@.data.mesh}\" | sed -e \"s@disablePolicyChecks: true@disablePolicyChecks: false@\" > /tmp/mesh.yaml")
	util.ShellMuteOutput("oc -n " + meshNamespace + " create cm istio -o yaml --dry-run --from-file=mesh=/tmp/mesh.yaml | oc replace -f -")
	log.Info("Verify disablePolicyChecks should be false")
	util.Shell("oc -n " + meshNamespace + " get cm istio -o jsonpath=\"{@.data.mesh}\" | grep disablePolicyChecks")

	t.Run("rate_limits_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.KubeApply(testNamespace, bookinfoAllv1Yaml, kubeconfigFile)
		util.KubeApplyContents(meshNamespace, rateLimitYaml, kubeconfigFile)
		
		log.Info("Sleep 90 seconds...")
		time.Sleep(time.Duration(90) * time.Second)

		log.Info("productpage permits 2 requests every 5 seconds. Verify 'Quota is exhausted' message")
		for i := 0; i < 40; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			defer util.CloseResponseBody(resp)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			if i == 0 {
				util.Inspect(
					util.CompareHTTPResponse(body, "productpage-normal-user-v1.html"),
					"Didn't get expected response.",
					"Success. Response matches with expected.",
					t)

			} else if i < 39{
				err = util.CompareHTTPResponse(body, "productpage-quota-exhausted.html")
				if err != nil {
					continue
				}

				util.Inspect(
					util.CompareHTTPResponse(body, "productpage-quota-exhausted.html"),
					"Didn't get quot exhausted message.",
					"Success. Response matches with expected.",
					t)
				break
			} else {
				util.Inspect(
					util.CompareHTTPResponse(body, "productpage-quota-exhausted.html"),
					"Didn't get quot exhausted message.",
					"Success. Response matches with expected.",
					t)
			}
		}
	})

	t.Run("conditional_rate_limits_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Conditional rate limits")
		util.KubeApply(meshNamespace, rateLimitConditionalYaml, kubeconfigFile)
		
		log.Info("Sleep 50 seconds...")
		time.Sleep(time.Duration(50) * time.Second)

		log.Info("productpage permits 2 requests every 5 seconds. Verify 'Quota is exhausted' message. Login user jason should not see quota message")
		
		for i := 0; i < 5; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, testUserJar)
			defer util.CloseResponseBody(resp)
			if i > 1 {
				util.Inspect(err, "failed to get HTTP Response", "", t)
				body, err := ioutil.ReadAll(resp.Body)
				util.Inspect(err, "failed to read response body", "", t)
				util.Inspect(
					util.CompareHTTPResponse(body, "productpage-test-user-v1.html"),
					"Didn't get expected response.",
					"Success. Response matches with expected.",
					t)
			}
		}
		
		util.GetHTTPResponse(fmt.Sprintf("http://%s/logout", ingress), testUserJar)
		time.Sleep(time.Duration(10) * time.Second)

		for i := 0; i < 20; i++ {
			resp, _, err := util.GetHTTPResponse(productpageURL, nil)
			defer util.CloseResponseBody(resp)
			util.Inspect(err, "failed to get HTTP Response", "", t)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)
			if i < 19 {
				err = util.CompareHTTPResponse(body, "productpage-quota-exhausted.html")
				if err != nil {
					continue
				}

				util.Inspect(
					util.CompareHTTPResponse(body, "productpage-quota-exhausted.html"),
					"Didn't get quot exhausted message.",
					"Success. Response matches with expected.",
					t)
				break
			} else {
				util.Inspect(
					util.CompareHTTPResponse(body, "productpage-quota-exhausted.html"),
					"Didn't get quot exhausted message.",
					"Success. Response matches with expected.",
					t)
			}
		}
	})

}
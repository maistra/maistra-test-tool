// Copyright 2020 Red Hat, Inc.
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

package tests

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupControlHeadersRouting(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(meshNamespace, keyvalRule, kubeconfig)
	time.Sleep(time.Duration(waitTime*6) * time.Second)
	util.KubeDeleteContents(meshNamespace, keyvalHandler, kubeconfig)
	util.KubeDeleteContents(meshNamespace, keyvalInstance, kubeconfig)
	util.KubeDelete(meshNamespace, keyvaltemplate, kubeconfig)
	util.KubeDelete(meshNamespace, keyvalYaml, kubeconfig)
	util.ShellMuteOutput("kubectl delete service keyval -n %s", meshNamespace)
	util.ShellMuteOutput("kubectl delete pod keyval -n %s", meshNamespace)
	util.KubeDeleteContents(namespace, httpbinGateway3, kubeconfig)
	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestControlHeadersRouting(t *testing.T) {
	defer cleanupControlHeadersRouting(testNamespace)
	defer recoverPanic(t)

	log.Info("Control Headers and Routing")

	log.Info("Enabling Mixer Plugins")
	util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{%s}'`,
		meshNamespace, smcpName,
		`"spec":{"policy":{"type": "Mixer", "mixer":{"enableChecks":true}}}`)

	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

	deployHttpbin(testNamespace)
	if err := util.KubeApplyContents(testNamespace, httpbinGateway3, kubeconfig); err != nil {
		t.Errorf("Failed to configure Gateway")
		log.Errorf("Failed to configure Gateway")
	}

	log.Info("Output Producing Adapters")
	util.Shell("kubectl run keyval --image=%s --namespace %s --port 9070 --expose", keyvalImage, meshNamespace)
	util.CheckPodRunning(meshNamespace, "run=keyval", kubeconfig)
	util.KubeApply(meshNamespace, keyvaltemplate, kubeconfig)
	util.KubeApply(meshNamespace, keyvalYaml, kubeconfig)
	util.KubeApplyContents(meshNamespace, keyvalHandler, kubeconfig)
	util.KubeApplyContents(meshNamespace, keyvalInstance, kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)

	t.Run("Policies_request_header_operations", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Request header operations")
		resp, _, err := util.GetHTTPResponse(fmt.Sprintf("http://%s/headers", gatewayHTTP), nil)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		log.Infof("%v", string(body))
		util.CloseResponseBody(resp)
	})

	t.Run("Policies_request_header_user_group_operations", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents(meshNamespace, keyvalRule, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 70 seconds...")
		time.Sleep(time.Duration(waitTime*15) * time.Second)

		log.Info(`Verify that user:json has "User-Group": "admin" header`)
		resp, err := checkUserGroup(fmt.Sprintf("http://%s/headers", gatewayHTTP), gatewayHTTP, ingressHTTPPort, "jason")
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		_, err = regexp.MatchString(`"User-Group":*"admin"`, string(body))
		if err == nil {
			log.Infof("Get expected headers: %s", string(body))
		} else {
			t.Errorf("Missing User-Group: admin headers: %s", string(body))
			log.Errorf("Missing User-Group: admin headers: %s", string(body))
		}
		util.CloseResponseBody(resp)
	})

	t.Run("Policies_request_header_418_operations", func(t *testing.T) {
		defer recoverPanic(t)

		util.KubeApplyContents(meshNamespace, keyvalRule418, kubeconfig)
		time.Sleep(time.Duration(waitTime*15) * time.Second)

		log.Info("Verify response 418 teapot")
		resp, err := checkUserGroup(fmt.Sprintf("http://%s/headers", gatewayHTTP), gatewayHTTP, ingressHTTPPort, "jason")
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		body, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		if strings.Contains(string(body), "teapot") {
			log.Infof("Get expected headers: %s", string(body))
		} else {
			t.Errorf("Expect teapot response. Got unexpected response: %s", string(body))
			log.Errorf("Expect teapot response. Got unexpected response: %s", string(body))
		}
		util.CloseResponseBody(resp)
	})
}

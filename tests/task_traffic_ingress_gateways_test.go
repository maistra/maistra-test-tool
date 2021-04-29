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
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupIngressGateways(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, httpbinGateway1, kubeconfig)
	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

}

func TestIngressGateways(t *testing.T) {
	defer cleanupIngressGateways(testNamespace)
	defer recoverPanic(t)

	log.Infof("# TestIngressGateways")
	deployHttpbin(testNamespace)

	if err := util.KubeApplyContents(testNamespace, httpbinGateway1, kubeconfig); err != nil {
		t.Errorf("Failed to configure Gateway")
		log.Errorf("Failed to configure Gateway")
	}
	time.Sleep(time.Duration(waitTime*4) * time.Second)

	t.Run("TrafficManagement_ingress_status_200_test", func(t *testing.T) {
		defer recoverPanic(t)

		resp, err := util.GetWithHost(fmt.Sprintf("http://%s/status/200", gatewayHTTP), "httpbin.example.com")
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)
	})

	t.Run("TrafficManagement_ingress_headers_test", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApplyContents(testNamespace, httpbinGateway2, kubeconfig); err != nil {
			t.Errorf("Failed to configure Gateway")
			log.Errorf("Failed to configure Gateway")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		resp, duration, err := util.GetHTTPResponse(fmt.Sprintf("http://%s/headers", gatewayHTTP), nil)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		log.Infof("httpbin headers page returned in %d ms", duration)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)
	})
}

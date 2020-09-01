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
	"maistra/util"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanupIngressK8s(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(meshNamespace, httpbinOCPRoute, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinIngress, kubeconfig)
	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestIngressK8s(t *testing.T) {
	defer cleanupIngressK8s(testNamespace)
	defer recoverPanic(t)

	log.Info("# Test Ingress Kubernetes")
	deployHttpbin(testNamespace)
	// OCP4 Route
	util.KubeApplyContents(meshNamespace, httpbinOCPRoute, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

	t.Run("TrafficManagement_ingress_k8s_test", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApplyContents(testNamespace, httpbinIngress, kubeconfig); err != nil {
			t.Errorf("Failed to configure Ingress")
			log.Errorf("Failed to configure Ingress")
		}
		resp, err := util.GetWithHost(fmt.Sprintf("http://%s/status/200", gatewayHTTP), "httpbin.example.com")
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		if resp.StatusCode != 200 {
			log.Errorf("Got unexpected response. status code is %d", resp.StatusCode)
			t.Errorf("Got unexpected response. status code is %d", resp.StatusCode)
		}

		resp, duration, err := util.GetHTTPResponse(fmt.Sprintf("http://%s/headers", gatewayHTTP), nil)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get HTTP Response", "", t)
		log.Infof("httpbin headers page returned in %d ms", duration)
		if resp.StatusCode != 404 {
			log.Errorf("Got unexpected response. status code is %d", resp.StatusCode)
			t.Errorf("Got unexpected response. status code is %d", resp.StatusCode)
		}
	})

	// TLS specifying IngressClass
}

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
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup08(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	OcDelete("", httpbinOCPRouteYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinRouteYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
}

func configHttpbin(namespace, kubeconfig string) error {
	if err := util.KubeApply(namespace, httpbinGatewayYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, httpbinRouteYaml, kubeconfig); err != nil {
		return err
	}
	if err := OcApply("", httpbinOCPRouteYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func updateHttpbin(namespace, kubeconfig string) error {
	log.Infof("# Update Httpbin")
	if err := util.KubeApply(namespace, httpbinGatewayv2Yaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, httpbinRoutev2Yaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func Test08 (t *testing.T) {
	log.Infof("# TC_08 Control Ingress Traffic")
	ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	Inspect(err, "failed to get ingressgateway URL", "", t)

	Inspect(deployHttpbin(testNamespace, kubeconfigFile), "failed to deploy httpbin", "", t)
	Inspect(configHttpbin(testNamespace, kubeconfigFile), "failed to config httpbin", "", t)

	t.Run("status_200", func(t *testing.T) {
		resp, err := GetWithHost(fmt.Sprintf("http://%s/status/200", ingress), "httpbin.example.com")
		defer CloseResponseBody(resp)
		Inspect(err, "failed to get response", "", t)
		Inspect(CheckHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
	})
	
	t.Run("headers", func(t *testing.T) {
		Inspect(updateHttpbin(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		resp, duration, err := GetHTTPResponse(fmt.Sprintf("http://%s/headers", ingress), nil)
		defer CloseResponseBody(resp)
		Inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("httpbin headers page returned in %d ms", duration)
		Inspect(CheckHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
	})
	defer cleanup08(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test failed: %v", err)
		}
	}()
}

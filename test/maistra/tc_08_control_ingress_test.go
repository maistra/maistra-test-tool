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
	"maistra/util"
)

func cleanup08(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.OcDelete(meshNamespace, httpbinOCPRouteYaml, kubeconfig) // uncomment this OcDelete when IOR is not enabled
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
	
	util.OcApply(meshNamespace, httpbinOCPRouteYaml, kubeconfig)   // uncomment this OcApply when IOR is not enabled

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

func Test08(t *testing.T) {
	defer cleanup08(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_08 Control Ingress Traffic")
	ingress, err := util.GetOCPIngressgateway("app=istio-ingressgateway", meshNamespace, kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)

	util.Inspect(deployHttpbin(testNamespace, kubeconfigFile), "failed to deploy httpbin", "", t)
	util.Inspect(configHttpbin(testNamespace, kubeconfigFile), "failed to config httpbin", "", t)
	log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)

	t.Run("status_200_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		resp, err := util.GetWithHost(fmt.Sprintf("http://%s/status/200", ingress), "httpbin.example.com")
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get response", "", t)
		util.Inspect(util.CheckHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
	})

	t.Run("headers_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		util.Inspect(updateHttpbin(testNamespace, kubeconfigFile), "failed to apply rules", "", t)
		resp, duration, err := util.GetHTTPResponse(fmt.Sprintf("http://%s/headers", ingress), nil)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("httpbin headers page returned in %d ms", duration)
		util.Inspect(util.CheckHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
	})

}

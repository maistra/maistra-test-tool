// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra

import (
	"fmt"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup07(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	OcDelete("", httpbinOCPRouteYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinRouteYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func deployHttpbin(namespace, kubeconfig string) error {
	log.Infof("# Deploy Httpbin")
	if err := util.KubeApply(namespace, httpbinYaml, kubeconfig); err != nil {
		return err
	}
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
	if err := util.CheckPodRunning(testNamespace, "app=httpbin", ""); err != nil {
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

func Test07 (t *testing.T) {
	log.Infof("# TC_07 Control Ingress Traffic")
	inspect(deployHttpbin(testNamespace, ""), "failed to deploy httpbin", "", t)

	t.Run("A1", func(t *testing.T) {
		resp, err := getWithHost(fmt.Sprintf("http://%s/status/200", host), "httpbin.example.com")
		defer closeResponseBody(resp)
		inspect(err, "failed to get response", "", t)
		inspect(checkHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
	})
	
	t.Run("A2", func(t *testing.T) {
		inspect(updateHttpbin(testNamespace, ""), "failed to apply rules", "", t)
		resp, duration, err := getHTTPResponse(fmt.Sprintf("http://%s/headers", host), nil)
		defer closeResponseBody(resp)
		inspect(err, "failed to get HTTP Response", "", t)
		log.Infof("httpbin headers page returned in %d ms", duration)
		inspect(checkHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
	})

	defer cleanup07(testNamespace, "")
}

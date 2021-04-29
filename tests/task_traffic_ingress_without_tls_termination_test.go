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
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupIngressWithOutTLS(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, nginxIngressGateway, kubeconfig)
	util.KubeDeleteContents(namespace, nginxServer, kubeconfig)
	util.ShellMuteOutput("kubectl delete secret nginx-server-certs -n %s", namespace)
	util.ShellMuteOutput("kubectl delete configmap nginx-configmap -n %s", namespace)
	util.Shell("kubectl get secret -n %s", namespace)
	util.Shell("kubectl get configmap -n %s", namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

}

func TestIngressWithOutTLS(t *testing.T) {
	defer cleanupIngressWithOutTLS(testNamespace)
	defer recoverPanic(t)

	log.Infof("# TestIngressWithOutTLS Termination")
	log.Info("Create Secret")
	if _, err := util.CreateTLSSecret("nginx-server-certs", testNamespace, nginxServerCertKey, nginxServerCert, kubeconfig); err != nil {
		t.Errorf("Failed to create secret %s\n", "nginx-server-certs")
		log.Infof("Failed to create secret %s\n", "nginx-server-certs")
	}

	log.Info("Create ConfigMap")
	util.Shell("kubectl create configmap nginx-configmap --from-file=nginx.conf=%s -n %s", "config/nginx.conf", testNamespace)

	log.Info("Deploy NGINX server")
	if err := util.KubeApplyContents(testNamespace, nginxServer, kubeconfig); err != nil {
		t.Errorf("Failed to deploy NGINX server")
		log.Errorf("Failed to deploy NGINX server")
	}
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(testNamespace, "run=my-nginx", kubeconfig)

	log.Info("Verify NGINX server")
	pod, err := util.GetPodName(testNamespace, "run=my-nginx", kubeconfig)
	cmd := fmt.Sprintf(`curl -v -k --resolve nginx.example.com:443:127.0.0.1 https://nginx.example.com | grep "Welcome to nginx"`)
	msg, err := util.PodExec(testNamespace, pod, "istio-proxy", cmd, true, kubeconfig)
	util.Inspect(err, "failed to get response", "", t)
	if !strings.Contains(msg, "Welcome to nginx") {
		t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
	} else {
		log.Infof("Success. Get expected response: %s", msg)
	}

	t.Run("TrafficManagement_ingress_configure_ingress_gateway_without_TLS_Termination", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Configure an ingress gateway")
		if err := util.KubeApplyContents(testNamespace, nginxIngressGateway, kubeconfig); err != nil {
			t.Errorf("Failed to configure NGINX ingress gateway")
			log.Errorf("Failed to configure NGINX ingress gateway")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		url := "https://nginx.example.com:" + secureIngressPort
		resp, err := curlWithCA(url, gatewayHTTP, secureIngressPort, "nginx.example.com", nginxServerCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "Welcome to nginx") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get Welcome to nginx: %v", string(bodyByte))
		}
	})
}

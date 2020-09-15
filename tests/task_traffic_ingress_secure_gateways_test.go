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
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupIngressTLSGateways(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(meshNamespace, httpbinTLSGatewayMTLS, kubeconfig)
	util.KubeDeleteContents(meshNamespace, multiHostsGateway, kubeconfig)
	util.KubeDeleteContents(meshNamespace, helloworldv1OCPRouteHTTPS, kubeconfig)
	util.KubeDeleteContents(meshNamespace, helloworldv1, kubeconfig)
	util.KubeDeleteContents(meshNamespace, httpbinOCPRouteHTTPS, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinGatewayHTTPS, kubeconfig)
	util.ShellMuteOutput("kubectl delete secret %s -n %s", "httpbin-credential", meshNamespace)
	util.ShellMuteOutput("kubectl delete secret %s -n %s", "helloworld-credential", meshNamespace)
	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*6) * time.Second)
}

func TestIngressTLSGateways(t *testing.T) {
	defer cleanupIngressTLSGateways(testNamespace)
	defer recoverPanic(t)

	log.Infof("# TestIngressTLSGateways")
	deployHttpbin(testNamespace)

	if _, err := util.CreateTLSSecret("httpbin-credential", meshNamespace, httpbinSampleServerCertKey, httpbinSampleServerCert, kubeconfig); err != nil {
		t.Errorf("Failed to create secret %s\n", "httpbin-credential")
		log.Infof("Failed to create secret %s\n", "httpbin-credential")
	}

	// config https gateway
	if err := util.KubeApplyContents(testNamespace, httpbinTLSGatewayHTTPS, kubeconfig); err != nil {
		t.Errorf("Failed to configure Gateway")
		log.Errorf("Failed to configure Gateway")
	}

	// OCP4 Route
	util.KubeApplyContents(meshNamespace, httpbinOCPRouteHTTPS, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

	t.Run("TrafficManagement_ingress_general_tls_test", func(t *testing.T) {
		defer recoverPanic(t)

		// check teapot
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := curlWithCA(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("TrafficManagement_ingress_multiple_hosts_tls_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Configure multiple hosts Gateway")
		util.ShellMuteOutput("kubectl delete secret %s -n %s", "httpbin-credential", meshNamespace)
		if _, err := util.CreateTLSSecret("httpbin-credential", meshNamespace, httpbinSampleServerCertKey, httpbinSampleServerCert, kubeconfig); err != nil {
			t.Errorf("Failed to create secret %s\n", "httpbin-credential")
			log.Infof("Failed to create secret %s\n", "httpbin-credential")
		}
		util.KubeApplyContents(meshNamespace, helloworldv1, kubeconfig)
		// OCP4 Route
		util.KubeApplyContents(meshNamespace, helloworldv1OCPRouteHTTPS, kubeconfig)
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		if _, err := util.CreateTLSSecret("helloworld-credential", meshNamespace, helloworldServerCertKey, helloworldServerCert, kubeconfig); err != nil {
			t.Errorf("Failed to create secret %s\n", "helloworld-credential ")
			log.Infof("Failed to create secret %s\n", "helloworld-credential ")
		}

		util.KubeApplyContents(meshNamespace, multiHostsGateway, kubeconfig)
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		url := "https://helloworld-v1.example.com:" + secureIngressPort + "/hello"
		resp, err := curlWithCA(url, gatewayHTTP, secureIngressPort, "helloworld-v1.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)

		// check teapot
		url = "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err = curlWithCA(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("TrafficManagement_ingress_mutual_tls_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Configure Mutual TLS Gateway")
		util.ShellMuteOutput("kubectl delete secret %s -n %s", "httpbin-credential", meshNamespace)
		// create ca secret
		_, err := util.ShellMuteOutput("kubectl create secret generic %s --from-file=tls.key=%s --from-file=tls.crt=%s --from-file=ca.crt=%s -n %s --kubeconfig=%s",
			"httpbin-credential", httpbinSampleServerCertKey, httpbinSampleServerCert, httpbinSampleCACert, meshNamespace, kubeconfig)
		if err != nil {
			log.Infof("Failed to create secret %s\n", "httpbin-credential")
			t.Errorf("Failed to create secret %s\n", "httpbin-credential")
		}

		// config mutual tls
		if err := util.KubeApplyContents(testNamespace, httpbinTLSGatewayMTLS, kubeconfig); err != nil {
			t.Errorf("Failed to configure Gateway")
			log.Errorf("Failed to configure Gateway")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		log.Info("Check SSL handshake failure as expected")
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := curlWithCA(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		if err != nil {
			log.Infof("Expected failure: %v", err)
		} else {
			bodyByte, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)

			t.Errorf("Unexpected response: %s", string(bodyByte))
			util.CloseResponseBody(resp)
		}

		log.Info("Check SSL return a teapot again")
		resp, err = curlWithCAClient(url, gatewayHTTP, secureIngressPort, "httpbin.example.com",
			httpbinSampleCACert, httpbinSampleClientCert, httpbinSampleClientCertKey)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			log.Info(string(bodyByte))
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})
}

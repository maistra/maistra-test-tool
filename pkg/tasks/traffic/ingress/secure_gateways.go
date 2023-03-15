// Copyright 2021 Red Hat, Inc.
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

package ingress

import (
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func cleanupSecureGateways() {
	log.Log.Info("Cleanup")
	httpbin := examples.Httpbin{"bookinfo"}
	util.KubeDeleteContents("bookinfo", httpbinTLSGatewayMTLS)
	util.KubeDeleteContents("bookinfo", multiHostsGateway)
	util.KubeDeleteContents("bookinfo", httpbinTLSGatewayHTTPS)
	util.ShellMuteOutput(`kubectl delete secret %s -n %s`, "httpbin-credential", meshNamespace)
	util.ShellMuteOutput(`kubectl delete secret %s -n %s`, "helloworld-credential", meshNamespace)
	util.KubeDeleteContents("bookinfo", helloworldv1)
	httpbin.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestSecureGateways(t *testing.T) {
	defer cleanupSecureGateways()
	defer util.RecoverPanic(t)

	log.Log.Info("Test Secure Gateways")
	httpbin := examples.Httpbin{"bookinfo"}
	httpbin.Install()

	if util.Getenv("SAMPLEARCH", "x86") == "p" {
		util.KubeApplyContents("bookinfo", helloworldv1P)
	} else if util.Getenv("SAMPLEARCH", "x86") == "z" {
		util.KubeApplyContents("bookinfo", helloworldv1Z)
	} else {
		util.KubeApplyContents("bookinfo", helloworldv1)
	}
	util.CheckPodRunning("bookinfo", "app=helloworld-v1")
	time.Sleep(time.Duration(10) * time.Second)

	log.Log.Info("Create TLS secrets")
	if _, err := util.CreateTLSSecret("httpbin-credential", meshNamespace, httpbinSampleServerCertKey, httpbinSampleServerCert); err != nil {
		t.Errorf("Failed to create secret %s\n", "httpbin-credential")
		log.Log.Infof("Failed to create secret %s\n", "httpbin-credential")
	}
	if _, err := util.CreateTLSSecret("helloworld-credential", meshNamespace, helloworldServerCertKey, helloworldServerCert); err != nil {
		t.Errorf("Failed to create secret %s\n", "helloworld-credential ")
		log.Log.Infof("Failed to create secret %s\n", "helloworld-credential ")
	}
	time.Sleep(time.Duration(10) * time.Second)

	t.Run("TrafficManagement_ingress_single_host_tls_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Configure a TLS ingress gateway for a single host")
		// config https gateway
		if err := util.KubeApplyContents("bookinfo", httpbinTLSGatewayHTTPS); err != nil {
			t.Errorf("Failed to configure Gateway")
			log.Log.Errorf("Failed to configure Gateway")
		}
		time.Sleep(time.Duration(30) * time.Second)

		// check teapot
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := util.CurlWithCA(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("TrafficManagement_ingress_multiple_hosts_tls_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Configure multiple hosts Gateway")
		if err := util.KubeApplyContents("bookinfo", multiHostsGateway); err != nil {
			t.Errorf("Failed to configure multihosts Gateway")
			log.Log.Errorf("Failed to configure multihosts Gateway")
		}
		time.Sleep(time.Duration(30) * time.Second)

		log.Log.Info("Check helloworld")
		url := "https://helloworld-v1.example.com:" + secureIngressPort + "/hello"
		resp, err := util.CurlWithCA(url, gatewayHTTP, secureIngressPort, "helloworld-v1.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		util.Inspect(util.CheckHTTPResponse200(resp), "Failed to get HTTP 200", resp.Status, t)

		log.Log.Info("Check teapot")
		url = "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err = util.CurlWithCA(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("TrafficManagement_ingress_mutual_tls_test", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Configure Mutual TLS Gateway")
		util.ShellMuteOutput(`kubectl delete secret %s -n %s`, "httpbin-credential", meshNamespace)
		// create ca secret
		_, err := util.ShellMuteOutput(`kubectl create secret generic %s --from-file=tls.key=%s --from-file=tls.crt=%s --from-file=ca.crt=%s -n %s`,
			"httpbin-credential", httpbinSampleServerCertKey, httpbinSampleServerCert, httpbinSampleCACert, meshNamespace)
		if err != nil {
			log.Log.Infof("Failed to create generic secret %s\n", "httpbin-credential")
			t.Errorf("Failed to generic create secret %s\n", "httpbin-credential")
		}
		time.Sleep(time.Duration(10) * time.Second)

		// config mutual tls
		if err := util.KubeApplyContents("bookinfo", httpbinTLSGatewayMTLS); err != nil {
			t.Errorf("Failed to configure Gateway")
			log.Log.Errorf("Failed to configure Gateway")
		}
		time.Sleep(time.Duration(30) * time.Second)

		log.Log.Info("Check SSL handshake failure as expected")
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := util.CurlWithCA(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		if err != nil {
			log.Log.Infof("Expected failure: %v", err)
		} else {
			bodyByte, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)

			t.Errorf("Unexpected response: %s", string(bodyByte))
			util.CloseResponseBody(resp)
		}

		log.Log.Info("Check SSL return a teapot again")
		resp, err = util.CurlWithCAClient(url, gatewayHTTP, secureIngressPort, "httpbin.example.com",
			httpbinSampleCACert, httpbinSampleClientCert, httpbinSampleClientCertKey)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Log.Info(string(bodyByte))
		} else {
			log.Log.Info(string(bodyByte))
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})
}

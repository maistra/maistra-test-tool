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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanupIngressHttps(namespace string) {
	log.Info("# Cleanup ...")

	util.KubeDeleteContents(meshNamespace, httpbinOCPRouteHTTPS, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinGatewayHTTPS, kubeconfig)	
	util.ShellMuteOutput("kubectl delete secret %s -n %s", "istio-ingressgateway-certs", meshNamespace)

	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)

}

func checkTeapot(url, ingressHost, secureIngressPort, host, cacertFile string) (*http.Response, error) {
	// Load CA cert
	caCert, err := ioutil.ReadFile(cacertFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS transport
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	// Custom DialContext
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == host+":"+secureIngressPort {
			addr = ingressHost + ":" + secureIngressPort
		}
		return dialer.DialContext(ctx, network, addr)
	}

	// Setup HTTPS client
	client := &http.Client{Transport: transport}

	// GET something
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Set host
	req.Host = host
	req.Header.Set("Host", req.Host)
	// Get response
	return client.Do(req)
}


func TestIngressHttps(t *testing.T) {
	defer cleanupIngressHttps(testNamespace)
	defer recoverPanic(t)

	log.Infof("# Securing Gateways with HTTPS")
	deployHttpbin(testNamespace)

	if _, err := util.CreateTLSSecret("istio-ingressgateway-certs", meshNamespace, httpbinSampleServerCertKey, httpbinSampleServerCert, kubeconfig); err != nil {
		t.Errorf("Failed to create secret %s\n", "istio-ingressgateway-certs")
		log.Infof("Failed to create secret %s\n", "istio-ingressgateway-certs")
	}

	// check cert
	pod, err := util.GetPodName(meshNamespace, "istio=ingressgateway", kubeconfig)
	msg, err := util.ShellSilent("kubectl exec --kubeconfig=%s -i -n %s %s -- %s ",
		kubeconfig, meshNamespace, pod, "ls -al /etc/istio/ingressgateway-certs | grep tls.crt")
	for err != nil {
		msg, err = util.ShellSilent("kubectl exec --kubeconfig=%s -i -n %s %s -- %s ",
			kubeconfig, meshNamespace, pod, "ls -al /etc/istio/ingressgateway-certs | grep tls.crt")
		time.Sleep(time.Duration(10) * time.Second)
	}
	log.Infof("Secret %s created: %s\n", "istio-ingressgateway-certs", msg)

	// config https gateway
	if err := util.KubeApplyContents(testNamespace, httpbinGatewayHTTPS, kubeconfig); err != nil {
		t.Errorf("Failed to configure Gateway")
		log.Errorf("Failed to configure Gateway")
	}

	// OCP4 Route
	if err := util.KubeApplyContents(meshNamespace, httpbinOCPRouteHTTPS, kubeconfig); err != nil {
		t.Errorf("Failed to configure OCP Route")
		log.Errorf("Failed to configure OCP Route")
	}

	time.Sleep(time.Duration(waitTime*4) * time.Second)
	
	t.Run("General_tls_test", func(t *testing.T) {
		defer recoverPanic(t)

		// check teapot
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := checkTeapot(url, gatewayHTTP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
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


}

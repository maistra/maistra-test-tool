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

func cleanupIngressHTTPS(namespace string) {
	log.Info("# Cleanup ...")

	//util.KubeDeleteContents(meshNamespace, bookinfoOCPRouteHTTPS, kubeconfig)
	util.KubeDeleteContents(meshNamespace, httpbinOCPRouteHTTPS, kubeconfig)
	util.KubeDeleteContents(namespace, httpbinGatewayHTTPS, kubeconfig)	
	util.ShellMuteOutput("kubectl delete secret %s -n %s", "istio-ingressgateway-certs", meshNamespace)
	util.ShellMuteOutput("kubectl delete secret %s -n %s", "istio-ingressgateway-ca-certs", meshNamespace)
	//util.ShellMuteOutput("kubectl delete secret %s -n %s", "istio-ingressgateway-bookinfo-certs", meshNamespace)
	//cleanBookinfo(namespace)
	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

}

func curlWithCA(url, ingressHost, secureIngressPort, host, cacertFile string) (*http.Response, error) {
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

func curlWithCAClient(url, ingressHost, secureIngressPort, host, cacertFile, certFile, keyFile string) (*http.Response, error) {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(cacertFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS transport
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
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
	defer cleanupIngressHTTPS(testNamespace)
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
		time.Sleep(time.Duration(waitTime*2) * time.Second)
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

	t.Run("Mutual_tls_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Configure Mutual TLS Gateway")
		// create ca secret
		_, err := util.ShellMuteOutput("kubectl create secret generic %s --from-file %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-ca-certs", httpbinSampleCACert, meshNamespace, kubeconfig)
		if err != nil {
			log.Infof("Failed to create secret %s\n", "istio-ingressgateway-ca-certs")
			t.Errorf("Failed to create secret %s\n", "istio-ingressgateway-ca-certs")
		}
		// check ca chain
		pod, err := util.GetPodName(meshNamespace, "istio=ingressgateway", kubeconfig)
		msg, err := util.ShellSilent("kubectl exec -it -n %s %s -- %s ",
			meshNamespace, pod, "ls -al /etc/istio/ingressgateway-ca-certs | grep example.com.crt")
		for err != nil {
			msg, err = util.ShellSilent("kubectl exec -it -n %s %s -- %s ",
				meshNamespace, pod, "ls -al /etc/istio/ingressgateway-ca-certs | grep example.com.crt")
			time.Sleep(time.Duration(waitTime*2) * time.Second)
		}
		log.Infof("Secret %s created: %s\n", "istio-ingressgateway-ca-certs", msg)

		// config mutual tls
		if err := util.KubeApplyContents(testNamespace, httpbinGatewayMTLS, kubeconfig); err != nil {
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

	/*
	t.Run("Multiple_hosts_tls_test", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Configure multiple hosts Gateway")
		if _, err := util.CreateTLSSecret("istio-ingressgateway-bookinfo-certs", meshNamespace, bookinfoServerCertKey, bookinfoServerCert, kubeconfig); err != nil {
			t.Errorf("Failed to create secret %s\n", "istio-ingressgateway-bookinfo-certs")
			log.Infof("Failed to create secret %s\n", "istio-ingressgateway-bookinfo-certs")
		}

		// config https gateway


		// verify gateway
		msg, err = util.ShellSilent("kubectl exec -i -n %s $(kubectl -n %s get pods -l istio=ingressgateway -o jsonpath='{.items[0].metadata.name}') -- %s",
			meshNamespace, meshNamespace, "ls -al /etc/istio/ingressgateway-bookinfo-certs | grep tls.crt")
		for err != nil {
			msg, err = util.ShellSilent("kubectl exec -i -n %s $(kubectl -n %s get pods -l istio=ingressgateway -o jsonpath='{.items[0].metadata.name}') -- %s",
				meshNamespace, meshNamespace, "ls -al /etc/istio/ingressgateway-bookinfo-certs | grep tls.crt")
			time.Sleep(time.Duration(waitTime*2) * time.Second)
		}
		log.Infof("Secret %s created: %s\n", "istio-ingressgateway-bookinfo-certs", msg)

		// OCP4 Route
		if err = util.KubeApplyContents(meshNamespace, bookinfoOCPRouteHTTPS, kubeconfig); err != nil {
			t.Errorf("Failed to configure OCP Route")
			log.Errorf("Failed to configure OCP Route")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		// deploy bookinfo
		deployBookinfo(testNamespace, true)
		if err = util.KubeApplyContents(meshNamespace, bookinfoGatewayHTTPS, kubeconfig); err != nil {
			t.Errorf("Failed to configure bookinfo gateway https")
			log.Errorf("Failed to configure bookinfo gateway https")
		}
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		// send a request to bookinfo productpage
		log.Info("Check SSL bookinfo productpage")
		url := "https://bookinfo.com:" + secureIngressPort + "/productpage"
		resp, err := curlWithCA(url, gatewayHTTP, secureIngressPort, "bookinfo.com", bookinfoSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)
		if strings.Contains(string(bodyByte), "200") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("Failed to get productpage: %v", string(bodyByte))
			log.Info(string(bodyByte))
		}

		// verify httpbin.example.com
		log.Info("Check SSL return a teapot")
		url = "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err = curlWithCAClient(url, gatewayHTTP, secureIngressPort, "httpbin.example.com",
			httpbinSampleCACert, httpbinSampleClientCert, httpbinSampleClientCertKey)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "Failed to get response", "", t)
		bodyByte, err = ioutil.ReadAll(resp.Body)
		util.Inspect(err, "Failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			log.Info(string(bodyByte))
			t.Errorf("Failed to get teapot: %v", string(bodyByte))
		}
	})
	*/
}

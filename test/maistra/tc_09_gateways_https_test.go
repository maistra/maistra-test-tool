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

func cleanup09(namespace, kubeconfig string) {

	log.Infof("# Cleanup. Following error can be ignored...")
	util.OcDelete("", httpbinOCPRouteYaml, kubeconfig) // uncomment this OcDelete when IOR is not enabled
	util.KubeDelete(namespace, httpbinGatewayHTTPSMutualYaml, kubeconfig)
	util.OcDelete("", httpbinOCPRouteHTTPSYaml, kubeconfig) // uncomment this OcDelete when IOR is not enabled
	util.KubeDelete(namespace, httpbinRouteHTTPSYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayHTTPSYaml, kubeconfig)

	util.ShellMuteOutput("oc delete secret %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-bookinfo-certs", "istio-system", kubeconfig)
	util.ShellMuteOutput("oc delete secret %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-certs", "istio-system", kubeconfig)
	util.ShellMuteOutput("oc delete secret %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-ca-certs", "istio-system", kubeconfig)
	util.ShellMuteOutput("oc delete secret %s -n %s --kubeconfig=%s",
		"istio.istio-ingressgateway-service-account", "istio-system", kubeconfig)

	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

// configHttpbinHTTPS configures https certs
func configHttpbinHTTPS(namespace, kubeconfig string) error {
	// create tls certs
	if _, err := util.CreateTLSSecret("istio-ingressgateway-certs", "istio-system", httpbinSampleServerCertKey, httpbinSampleServerCert, kubeconfig); err != nil {
		log.Infof("Failed to create secret %s\n", "istio-ingressgateway-certs")
		return err
	}
	// check cert
	pod, err := util.GetPodName("istio-system", "istio=ingressgateway", kubeconfig)
	msg, err := util.ShellSilent("oc exec --kubeconfig=%s -it -n %s %s -- %s ",
		kubeconfig, "istio-system", pod, "ls -al /etc/istio/ingressgateway-certs | grep tls.crt")
	for err != nil {
		msg, err = util.ShellSilent("oc exec --kubeconfig=%s -it -n %s %s -- %s ",
			kubeconfig, "istio-system", pod, "ls -al /etc/istio/ingressgateway-certs | grep tls.crt")
		time.Sleep(time.Duration(10) * time.Second)
	}
	log.Infof("Secret %s created: %s\n", "istio-ingressgateway-certs", msg)

	// config https
	if err := util.KubeApply(namespace, httpbinGatewayHTTPSYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply(namespace, httpbinRouteHTTPSYaml, kubeconfig); err != nil {
		return err
	}
	
	util.OcApply("", httpbinOCPRouteHTTPSYaml, kubeconfig)   // uncomment this OcApply when IOR is not enabled

	log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
	return nil
}

func updateHttpbinHTTPS(namespace, kubeconfig string) error {
	// create secret ca
	_, err := util.ShellMuteOutput("oc create secret generic %s --from-file %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-ca-certs", httpbinSampleCACert, "istio-system", kubeconfig)
	if err != nil {
		log.Infof("Failed to create secret %s\n", "istio-ingressgateway-ca-certs")
		return err
	}
	// check ca chain
	pod, err := util.GetPodName("istio-system", "istio=ingressgateway", kubeconfig)
	msg, err := util.ShellSilent("oc exec --kubeconfig=%s -it -n %s %s -- %s ",
		kubeconfig, "istio-system", pod, "ls -al /etc/istio/ingressgateway-ca-certs | grep ca-chain")
	for err != nil {
		msg, err = util.ShellSilent("oc exec --kubeconfig=%s -it -n %s %s -- %s ",
			kubeconfig, "istio-system", pod, "ls -al /etc/istio/ingressgateway-ca-certs | grep ca-chain")
		time.Sleep(time.Duration(10) * time.Second)
	}
	log.Infof("Secret %s created: %s\n", "istio-ingressgateway-ca-certs", msg)

	// config mutual tls
	if err := util.KubeApply(namespace, httpbinGatewayHTTPSMutualYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)
	return nil
}

func configHTTPSBookinfo(namespace, kubeconfig string) error {
	// create tls certs
	if _, err := util.CreateTLSSecret("istio-ingressgateway-bookinfo-certs", "istio-system", bookinfoSampleServerCertKey, bookinfoSampleServerCert, kubeconfig); err != nil {
		log.Infof("Failed to create secret %s\n", "istio-ingressgateway-certs")
		return err
	}

	return nil
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

func checkTeapot2(url, ingressHost, secureIngressPort, host, cacertFile, certFile, keyFile string) (*http.Response, error) {
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

func Test09(t *testing.T) {
	defer cleanup09(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occurred. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_09 Securing Gateways with HTTPS")
	log.Info("Waiting for previous run to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	ingressHost, err := util.GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	util.Inspect(err, "failed to get ingressgateway URL", "", t)

	secureIngressPort, err := util.GetSecureIngressPort("istio-system", "istio-ingressgateway", kubeconfigFile)
	util.Inspect(err, "cannot get ingress secure port", "", t)
	util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)

	util.Inspect(deployHttpbin(testNamespace, kubeconfigFile), "failed to deploy httpbin", "", t)
	util.Inspect(configHttpbinHTTPS(testNamespace, kubeconfigFile), "failed to config httpbin with tls certs", "", t)

	t.Run("general_tls_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		// check teapot
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := checkTeapot(url, ingressHost, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("mutual_tls_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occurred. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Configure Mutual TLS Gateway")
		util.Inspect(updateHttpbinHTTPS(testNamespace, kubeconfigFile), "failed to configure mutual tls gateway", "", t)

		log.Info("Check SSL handshake failure as expected")
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := checkTeapot(url, ingressHost, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		if err != nil {
			log.Infof("Expected failure: %v", err)
			// Don't need to close resp because resp is nil when err is not nil
		} else {
			bodyByte, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "failed to read response body", "", t)

			t.Errorf("Unexpected response: %s", string(bodyByte))
			util.CloseResponseBody(resp)
		}

		log.Info("Check SSL return a teapot again")
		resp, err = checkTeapot2(url, ingressHost, secureIngressPort, "httpbin.example.com",
			httpbinSampleCACert, httpbinSampleClientCert, httpbinSampleClientCertKey)
		defer util.CloseResponseBody(resp)
		util.Inspect(err, "failed to get response", "", t)

		bodyByte, err := ioutil.ReadAll(resp.Body)
		util.Inspect(err, "failed to read response body", "", t)

		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			log.Info(string(bodyByte))
			t.Errorf("failed to get teapot: %v", string(bodyByte))
		}
	})

	// configure TLS ingress gateway for multiple hosts
	// bookinfo TLS

}

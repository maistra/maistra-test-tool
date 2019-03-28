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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup09(namespace, kubeconfig string) {
	
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete("istio-system", jwtAuthYaml, kubeconfig)
	OcDelete("", httpbinOCPRouteYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayHTTPSMutualYaml, kubeconfig)
	OcDelete("", httpbinOCPRouteHTTPSYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinRouteHTTPSYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayHTTPSYaml, kubeconfig)
	
	util.ShellMuteOutput("kubectl delete secret %s -n %s --kubeconfig=%s", 
		"istio-ingressgateway-certs", "istio-system", kubeconfig)
	util.ShellMuteOutput("kubectl delete secret %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-ca-certs", "istio-system", kubeconfig)
	util.ShellMuteOutput("kubectl delete secret %s -n %s --kubeconfig=%s",
		"istio.istio-ingressgateway-service-account", "istio-system", kubeconfig)
	
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	
	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	cleanBookinfo(namespace, kubeconfig)
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
	msg, err := util.ShellSilent("kubectl exec --kubeconfig=%s -it -n %s %s -- %s ",
							kubeconfig, "istio-system", pod, "ls -al /etc/istio/ingressgateway-certs | grep tls.crt")
	for err != nil {
		msg, err = util.ShellSilent("kubectl exec --kubeconfig=%s -it -n %s %s -- %s ",
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
	if err := OcApply("", httpbinOCPRouteHTTPSYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
	return nil
}

func updateHttpbinHTTPS(namespace, kubeconfig string) error{
	// create secret ca
	_, err := util.ShellMuteOutput("kubectl create secret generic %s --from-file %s -n %s --kubeconfig=%s", 
									"istio-ingressgateway-ca-certs", httpbinSampleCACert , "istio-system", kubeconfig)
	if err != nil {
		log.Infof("Failed to create secret %s\n", "istio-ingressgateway-ca-certs")
		return err
	}
	// check ca chain
	pod, err := util.GetPodName("istio-system", "istio=ingressgateway", kubeconfig)
	msg, err := util.ShellSilent("kubectl exec --kubeconfig=%s -it -n %s %s -- %s ",
							kubeconfig, "istio-system", pod, "ls -al /etc/istio/ingressgateway-ca-certs | grep ca-chain")
	for err != nil {
		msg, err = util.ShellSilent("kubectl exec --kubeconfig=%s -it -n %s %s -- %s ",
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

func configJWT(kubeconfig string) error {
	// config jwt auth
	if err := OcApply("", httpbinOCPRouteYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("istio-system", jwtAuthYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func checkTeapot(url, ingressHostIP, secureIngressPort, host, cacertFile string) (*http.Response, error) {
	// Load CA cert
	caCert, err := ioutil.ReadFile(cacertFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS transport
	tlsConfig := &tls.Config{
		RootCAs:	caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	// Custom DialContext
	dialer := &net.Dialer{
		Timeout:		30 * time.Second,
		KeepAlive:		30 * time.Second,
		DualStack:		true,
	}

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == host + ":" + secureIngressPort {
			addr = ingressHostIP + ":" + secureIngressPort
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

func checkTeapot2(url, ingressHostIP, secureIngressPort, host, cacertFile, certFile, keyFile string) (*http.Response, error) {
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
		Certificates:		[]tls.Certificate{cert},
		RootCAs:			caCertPool,
	}
	tlsConfig.BuildNameToCertificate()
	transport := &http.Transport{TLSClientConfig: tlsConfig}

	// Custom DialContext
	dialer := &net.Dialer{
		Timeout:		30 * time.Second,
		KeepAlive:		30 * time.Second,
		DualStack:		true,
	}

	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == host + ":" + secureIngressPort {
			addr = ingressHostIP + ":" + secureIngressPort
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

func Test09 (t *testing.T) {
	defer cleanup09(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()
	panic("blocked by maistra-348")

	log.Infof("# TC_09 Securing Gateways with HTTPS")
	log.Info("Waiting for previous run to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)

	ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
	Inspect(err, "failed to get ingressgateway URL", "", t)
	
	ingressHostIP, err := GetIngressHostIP(kubeconfigFile)
	Inspect(err, "cannot get ingress host ip", "", t)
	
	secureIngressPort, err := GetSecureIngressPort("istio-system", "istio-ingressgateway", kubeconfigFile)
	Inspect(err, "cannot get ingress secure port", "", t)
	Inspect(deployBookinfo(testNamespace, kubeconfigFile, false), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
	
	Inspect(deployHttpbin(testNamespace, kubeconfigFile), "failed to deploy httpbin", "", t)
	Inspect(configHttpbinHTTPS(testNamespace, kubeconfigFile), "failed to config httpbin with tls certs", "", t)

	t.Run("general_tls", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		// check teapot
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := checkTeapot(url, ingressHostIP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		defer CloseResponseBody(resp)
		Inspect(err, "failed to get response", "", t)
		
		bodyByte, err := ioutil.ReadAll(resp.Body)
		Inspect(err, "failed to read response body", "", t)
		
		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("mutual_tls", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Configure Mutual TLS Gateway")
		Inspect(updateHttpbinHTTPS(testNamespace, kubeconfigFile), "failed to configure mutual tls gateway", "", t)
		
		log.Info("Check SSL handshake failure as expected")
		url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"
		resp, err := checkTeapot(url, ingressHostIP, secureIngressPort, "httpbin.example.com", httpbinSampleCACert)
		if err != nil {
			log.Infof("Expected failure: %v", err)
			// Don't need to close resp because resp is nil when err is not nil
		} else {
			bodyByte, err := ioutil.ReadAll(resp.Body)
			Inspect(err, "failed to read response body", "", t)
			
			t.Errorf("Unexpected response: %s", string(bodyByte))
			CloseResponseBody(resp)
		}
		
		log.Info("Check SSL return a teapot again")
		resp, err = checkTeapot2(url, ingressHostIP, secureIngressPort, "httpbin.example.com", 
									httpbinSampleCACert, httpbinSampleClientCert, httpbinSampleClientCertKey)
		defer CloseResponseBody(resp)
		Inspect(err, "failed to get response", "", t)
		
		bodyByte, err := ioutil.ReadAll(resp.Body)
		Inspect(err, "failed to read response body", "", t)
		
		if strings.Contains(string(bodyByte), "-=[ teapot ]=-") {
			log.Info(string(bodyByte))
		} else {
			t.Errorf("failed to get teapot: %v", string(bodyByte))
		}
	})

	t.Run("jwt", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()
		
		log.Info("Configure JWT Authentication")
		Inspect(configJWT(kubeconfigFile), "failed to configure JWT authentication", "", t)
		// check 401
		resp, err := GetWithHost(fmt.Sprintf("http://%s/status/200", ingress), "httpbin.example.com")
		Inspect(err, "failed to get response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Unexpected response code: %v", resp.StatusCode)
		} else {
			log.Info("Get expected response code: 401")
		}
		CloseResponseBody(resp)

		// check 200
		resp, err = http.Get(jwtURL)
		Inspect(err, "failed to get JWT response", "", t)
		
		tokenByte, err := ioutil.ReadAll(resp.Body)
		Inspect(err, "failed to read JWT response body", "", t)
		
		token := strings.Trim(string(tokenByte),"\n")
		CloseResponseBody(resp)

		resp, err = GetWithJWT(fmt.Sprintf("http://%s/status/200", ingress), token, "httpbin.example.com")
		Inspect(err, "failed to get response", "", t)
		Inspect(CheckHTTPResponse200(resp), "failed to get HTTP 200", "Get expected response code: 200", t)
		CloseResponseBody(resp)
	})
	
}

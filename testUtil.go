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
	"testing"
	"time"

	"maistra/util"
)

func recoverPanic(t *testing.T) {
	// recover from panic if one occurred. This allows cleanup to be executed after panic.
	if err := recover(); err != nil {
		t.Errorf("Test panic: %v", err)
	}
}

func isWithinPercentage(count int, total int, rate float64, tolerance float64) bool {
	minimum := int((rate - tolerance) * float64(total))
	maximum := int((rate + tolerance) * float64(total))
	return count >= minimum && count <= maximum
}

func prepareOCPConfig() {
	// create testing ns bookinfo, foo, bar, legacy
	util.ShellSilent("oc new-project bookinfo")
	util.ShellSilent("oc new-project foo")
	util.ShellSilent("oc new-project bar")
	util.ShellSilent("oc new-project legacy")
	time.Sleep(time.Duration(waitTime) * time.Second)

	// update smmr
	util.ShellSilent("kubectl apply -n %s -f %s", meshNamespace, "configFiles/smmrTest.yaml")
	time.Sleep(time.Duration(waitTime) * time.Second)

	// if testing in mtls disable mode, update mtls to false
	util.ShellSilent("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	// enable ior
	util.ShellSilent("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"gateways\":{\"istio-ingressgateway\":{\"ior_enabled\":\"true\"}}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)

	// TBD: path smcp ingressgateway loadbalancer

	// add anyuid
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z bookinfo-productpage -n %s", testNamespace)
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z bookinfo-reviews -n %s", testNamespace)
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z bookinfo-ratings-v2 -n %s", testNamespace)
	util.ShellSilent("oc adm policy add-scc-to-user anyuid -z default -n %s", testNamespace)
	time.Sleep(time.Duration(waitTime) * time.Second)
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

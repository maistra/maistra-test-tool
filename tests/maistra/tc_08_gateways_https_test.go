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
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)

func cleanup08(namespace, kubeconfig string) {
	if err := recover(); err != nil {
		log.Infof("Test failed: %v", err)
	}

	log.Infof("# Cleanup. Following error can be ignored...")
	OcDelete("", httpbinOCPRouteYaml, kubeconfig)
	OcDelete("", httpbinOCPRouteHTTPSYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayHTTPSYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinGatewayHTTPSMutualYaml, kubeconfig)
	util.KubeDelete(namespace, httpbinRouteHTTPSYaml, kubeconfig)

	util.ShellSilent("kubectl delete secret %s -n %s --kubeconfig=%s", 
		"istio-ingressgateway-certs", "istio-system", kubeconfig)
	util.ShellSilent("kubectl delete secret %s -n %s --kubeconfig=%s",
		"istio-ingressgateway-ca-certs", "istio-system", kubeconfig)
	util.ShellSilent("kubectl delete secret %s -n %s --kubeconfig=%s",
		"istio.istio-ingressgateway-service-account", "istio-system", kubeconfig)
	
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	
	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	util.KubeDelete("istio-system", jwtAuthYaml, kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}


// deployHttpbinHTTPS deploys httpbin and configures https certs
func deployHttpbinHTTPS(namespace, kubeconfig string) error {
	log.Infof("# Deploy Httpbin")
	if err := util.KubeApply(namespace, httpbinYaml, kubeconfig); err != nil {
		return err
	}
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

// TBD: there may be a better way to implement this curl in Go
func checkTeapot(ingressHostIP string) (string, error) {
	url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"

	msg, err := util.ShellMuteOutput("curl -v -HHost:%s --resolve %s:%s:%s --cacert %s %s",
							"httpbin.example.com", "httpbin.example.com",
							secureIngressPort,
							ingressHostIP,
							httpbinSampleCACert,
							url)
	
	if err != nil {
		return "curl command error", err
	}
	if !strings.Contains(msg, "SSL certificate verify ok") || !strings.Contains(msg, "-=[ teapot ]=-") {
		return msg, fmt.Errorf("error response")
	}
	return msg, nil
}

// TBD: there may be a better way to implement this curl in Go
func checkTeapot2(ingressHostIP string) (string, error) {
	url := "https://httpbin.example.com:" + secureIngressPort + "/status/418"

	msg, err := util.ShellMuteOutput("curl -HHost:%s --resolve %s:%s:%s --cacert %s --cert %s --key %s %s",
							"httpbin.example.com", "httpbin.example.com",
							secureIngressPort,
							ingressHostIP,
							httpbinSampleCACert,
							httpbinSampleClientCert,
							httpbinSampleClientCertKey,
							url)
	
	if err != nil {
		return "curl command error", err
	}
	if !strings.Contains(msg, "-=[ teapot ]=-") {
		return msg, fmt.Errorf("error response")
	}
	return msg, nil
}

func Test08 (t *testing.T) {
	log.Infof("# TC_08 Securing Gateways with HTTPS")

	// TBD maybe a better way to find the ip
	ingressHostIP, err := util.Shell("dig +short %s | tr -d '\n'", testIngressHostname)
	if err != nil {
		t.Errorf("cannot get ingress host ip: %v", err)
	}

	Inspect(deployHttpbinHTTPS(testNamespace, ""), "failed to deploy httpbin with tls certs", "", t)

	t.Run("general_tls", func(t *testing.T) {
		// check teapot
		msg, err := checkTeapot(ingressHostIP)
		Inspect(err, msg, msg, t)
	})

	t.Run("mutual_tls", func(t *testing.T) {
		log.Info("Configure Mutual TLS Gateway")
		Inspect(updateHttpbinHTTPS(testNamespace, ""), "failed to configure mutual tls gateway", "", t)
		
		log.Info("Check SSL handshake failure as expected")
		msg, err := checkTeapot(ingressHostIP)
		if err != nil {
			log.Infof("Expected failure: %s", msg)
		} else {
			t.Errorf("Unexpected response: %s", msg)
		}

		log.Info("Check SSL return a teapot again")
		msg, err = checkTeapot2(ingressHostIP)
		Inspect(err, msg, msg, t)
	})

	t.Run("jwt", func(t *testing.T) {
		log.Info("Configure JWT Authentication")
		Inspect(configJWT(""), "failed to configure JWT authentication", "", t)
		// check 401
		resp, err := GetWithHost(fmt.Sprintf("http://%s/status/200", ingressURL), "httpbin.example.com")
		Inspect(err, "failed to get response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Unexpected response code: %v", resp.StatusCode)
		} else {
			log.Info("Get expected response code: 401")
		}
		CloseResponseBody(resp)

		// check 200
		token, err := util.ShellMuteOutput("curl https://raw.githubusercontent.com/istio/istio/release-1.0/security/tools/jwt/samples/demo.jwt -s | tr -d '\n'")
		if err != nil {
			t.Error(err)
		}
		resp, err = GetWithJWT(fmt.Sprintf("http://%s/status/200", ingressURL), token, "httpbin.example.com")
		Inspect(err, "failed to get response", "", t)
		Inspect(CheckHTTPResponse200(resp), "failed to get HTTP 200", resp.Status, t)
		CloseResponseBody(resp)
	})
	defer cleanup08(testNamespace, "")
}
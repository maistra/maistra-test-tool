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
	"fmt"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupAuthMTLS(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete("foo", sleepYaml, kubeconfig)
	util.KubeDelete("foo", httpbinYaml, kubeconfig)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
}

func TestAuthMTLS(t *testing.T) {
	defer cleanupAuthMTLS(testNamespace)
	defer recoverPanic(t)

	log.Info("Mutual TLS Deep-Dive")

	// update mtls to true
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

	deployHttpbin("foo")
	deploySleep("foo")

	log.Info("Verify Citadel runs properly")
	util.Shell("kubectl get deploy -l istio=citadel -n %s", meshNamespace)

	t.Run("Security_authentication_verify_keys_certs", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Verify keys and certificates installation")
		httpbinPod, err := util.GetPodName("foo", "app=httpbin", kubeconfig)
		util.Inspect(err, "Failed to get httpbin pod name", "", t)
		cmd := fmt.Sprintf("ls /etc/certs")
		msg, err := util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfig)
		util.Inspect(err, "Failed to get certs", "", t)
		if strings.Contains(msg, "cert-chain.pem") && strings.Contains(msg, "key.pem") && strings.Contains(msg, "root-cert.pem") {
			log.Info("Success. All key and certs exist")
		} else {
			log.Errorf("Missing certs; Got this result: %s", msg)
			t.Errorf("Missing certs; Got this result: %s", msg)
		}

		log.Info("Check certificate is valid")
		cmd = fmt.Sprintf("cat /etc/certs/cert-chain.pem | openssl x509 -text -noout  | grep Validity -A 2")
		msg, err = util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfig)
		util.Inspect(err, "Error in grep certificate Validity", "", t)
		cmd = fmt.Sprintf("cat /etc/certs/cert-chain.pem | openssl x509 -text -noout  | grep 'Subject Alternative Name' -A 1")
		msg, err = util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfig)
		util.Inspect(err, "Error in grep certificate Validity", "", t)
	})

	t.Run("Security_authentication_verify_mtls_plain_text_requests", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Verify plain-text requests")
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl http://httpbin:8000/headers -o /dev/null -s -w '%%{http_code}'")
		msg, err := util.PodExec("foo", sleepPod, "istio-proxy", cmd, false, kubeconfig)
		if err != nil {
			log.Infof("Plain-text requests fail as expected: %s", msg)
		} else {
			log.Errorf("Unexpected response from plain-text requests: %s", msg)
			t.Errorf("Unexpected response from plain-text requests: %s", msg)
		}
	})

	t.Run("Security_authentication_verify_mtls_without_cert_requests", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Verify requests without client cert")
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://httpbin:8000/headers -o /dev/null -s -w '%%{http_code}' -k")
		msg, err := util.PodExec("foo", sleepPod, "istio-proxy", cmd, false, kubeconfig)
		if err != nil {
			log.Infof("Requests without client cert fail as expected: %s", msg)
		} else {
			log.Errorf("Unexpected response from requests without client cert: %s", msg)
			t.Errorf("Unexpected response from requests without client cert: %s", msg)
		}
	})

	t.Run("Security_authentication_verify_mtls_with_cert_requests", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Verify requests with client cert")
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://httpbin:8000/headers -o /dev/null -s -w '%%{http_code}' --key /etc/certs/key.pem --cert /etc/certs/cert-chain.pem --cacert /etc/certs/root-cert.pem -k")
		msg, err := util.PodExec("foo", sleepPod, "istio-proxy", cmd, false, kubeconfig)
		if strings.Contains(msg, "200") {
			log.Infof("Requests with client cert succeed: %s", msg)
		} else {
			log.Errorf("Unexpected response from requests with client cert: %s", msg)
			t.Errorf("Unexpected response from requests with client cert: %s", msg)
		}
	})
}

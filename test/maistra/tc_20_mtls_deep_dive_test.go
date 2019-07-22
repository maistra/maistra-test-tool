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
	"fmt"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup20(kubeconfig string) {
    log.Infof("# Cleanup. Following error can be ignored...")
    util.KubeDelete("foo", httpbinYaml, kubeconfig)
	util.KubeDelete("foo", sleepYaml, kubeconfig)
    util.ShellMuteOutput("oc delete meshpolicy default")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
    //util.DeleteNamespace("foo", kubeconfig)
}

func Test20(t *testing.T) {
	defer cleanup20(kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()
	
	log.Infof("# TC_20 Mutual TLS Deep-Dive")
	log.Info("Enable mTLS")
	util.Inspect(util.KubeApplyContents("", meshPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)	

	util.Inspect(util.CreateNamespace("foo", kubeconfigFile), "failed to create namespace", "", t)
	util.OcGrantPermission("default", "foo", kubeconfigFile)

	util.Inspect(deployHttpbin("foo", kubeconfigFile), "failed to deploy httpbin", "", t)
	util.Inspect(deploySleep("foo", kubeconfigFile), "failed to deploy sleep", "", t)

	t.Run("verify_citadel", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Verify Citadel runs properly. Available column should be 1 below")
		util.Shell("oc get deploy -l istio=citadel -n istio-system")
	})

	t.Run("verify_certs_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Verify keys and certs. cert-chain.pem, key.pem and root-cert.pem should be listed below")
		httpbinPod, err := util.GetPodName("foo", "app=httpbin", kubeconfigFile)
		util.Inspect(err, "failed to get httpbin pod name", "", t)
		cmd := fmt.Sprintf("ls /etc/certs")
		msg, err := util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfigFile)
		util.Inspect(err, "failed to get certs", "", t)
		if strings.Contains(msg, "cert-chain.pem") && strings.Contains(msg, "key.pem") && strings.Contains(msg, "root-cert.pem") {
			log.Info("Success. All key and certs exist")
		} else {
			log.Errorf("Missing certs; Got this result: %s", msg)
			t.Errorf("Missing certs; Got this result: %s", msg)
		}

		log.Info("Check certificate is valid")
		cmd = fmt.Sprintf("cat /etc/certs/cert-chain.pem | openssl x509 -text -noout  | grep Validity -A 2")
		msg, err = util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfigFile)
		util.Inspect(err, "Error in grep certificate Validity", "", t)
		cmd = fmt.Sprintf("cat /etc/certs/cert-chain.pem | openssl x509 -text -noout  | grep 'Subject Alternative Name' -A 1")
		msg, err = util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfigFile)
		util.Inspect(err, "Error in grep certificate Validity", "", t)
	})

	t.Run("request_plain_text_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Verify plain-text requests")
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl http://httpbin:8000/headers -o /dev/null -s -w '%%{http_code}'")
		msg, err := util.PodExec("foo", sleepPod, "istio-proxy", cmd, false, kubeconfigFile)
		if err != nil {
			log.Infof("plain-text requests fail as expected: %s", msg)
		} else {
			log.Errorf("Unexpected response from plain-text requests: %s", msg)
			t.Errorf("Unexpected response from plain-text requests: %s", msg)
		}
	})

	t.Run("request_without_cert_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Verify requests without client cert")
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://httpbin:8000/headers -o /dev/null -s -w '%%{http_code}' -k")
		msg, err := util.PodExec("foo", sleepPod, "istio-proxy", cmd, false, kubeconfigFile)
		if err != nil {
			log.Infof("requests without client cert fail as expected: %s", msg)
		} else {
			log.Errorf("Unexpected response from requests without client cert: %s", msg)
			t.Errorf("Unexpected response from requests without client cert: %s", msg)
		}
	})

	t.Run("request_with_cert_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Verify requests with client cert")
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://httpbin:8000/headers -o /dev/null -s -w '%%{http_code}' --key /etc/certs/key.pem --cert /etc/certs/cert-chain.pem --cacert /etc/certs/root-cert.pem -k")
		msg, err := util.PodExec("foo", sleepPod, "istio-proxy", cmd, false, kubeconfigFile)
		if strings.Contains(msg, "200") {
			log.Infof("requests with client cert succeed: %s", msg)
		} else {
			log.Errorf("Unexpected response from requests with client cert: %s", msg)
			t.Errorf("Unexpected response from requests with client cert: %s", msg)
		}
	})
}
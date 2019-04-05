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
	"istio.io/istio/tests/util"
)

func cleanup20(kubeconfig string) {
    log.Infof("# Cleanup. Following error can be ignored...")
    util.KubeDelete("foo", httpbinYaml, kubeconfig)
	util.KubeDelete("foo", sleepYaml, kubeconfig)
    util.ShellMuteOutput("kubectl delete meshpolicy default")
	log.Info("Waiting... Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
    util.DeleteNamespace("foo", kubeconfig)
}

func Test20(t *testing.T) {
	defer cleanup20(kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()
	panic("WIP")
	
	log.Infof("# TC_20 Mutual TLS Deep-Dive")
	log.Info("Enable mTLS")
	Inspect(util.KubeApplyContents("", meshPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
	log.Info("Waiting... Sleep 5 seconds...")
	time.Sleep(time.Duration(5) * time.Second)	

	Inspect(util.CreateNamespace("foo", kubeconfigFile), "failed to create namespace", "", t)
	OcGrantPermission("default", "foo", kubeconfigFile)

	Inspect(deployHttpbin("foo", kubeconfigFile), "failed to deploy httpbin", "", t)
	Inspect(deploySleep("foo", kubeconfigFile), "failed to deploy sleep", "", t)

	t.Run("verify_citadel", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Verify Citadel runs properly. Available column should be 1 below")
		util.Shell("kubectl get deploy -l istio=citadel -n istio-system")
	})

	t.Run("verify_certs", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("Verify keys and certs. cert-chain.pem, key.pem and root-cert.pem should be listed below")
		httpbinPod, err := util.GetPodName("foo", "app=httpbin", kubeconfigFile)
		Inspect(err, "failed to get httpbin pod name", "", t)
		cmd := fmt.Sprintf("ls /etc/certs")
		msg, err := util.PodExec("foo", httpbinPod, "istio-proxy", cmd, false, kubeconfigFile)
		Inspect(err, "failed to get certs", "", t)
		if strings.Contains(msg, "cert-chain.pem") && strings.Contains(msg, "key.pem") && strings.Contains(msg, "root-cert.pem") {
			log.Info("Success. All key and certs exist")
		} else {
			log.Errorf("Missing certs; Got this result: %s", msg)
			t.Errorf("Missing certs; Got this result: %s", msg)
		}
	})

	t.Run("validate_certs", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()


	})

	t.Run("check_mtls", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

	})

	t.Run("check_conflict", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

	})

	t.Run("request_plain_text", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

	})

	t.Run("request_without_cert", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

	})

	t.Run("request_with_cert", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

	})

}
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

func cleanup16(namespace, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete(namespace, sleepYaml, kubeconfig)
	util.KubeDelete(namespace, nginxYaml, kubeconfig)
	util.ShellMuteOutput("kubectl delete configmap nginxconfigmap -n %s --kubeconfig=%s", namespace, kubeconfig)
	util.ShellMuteOutput("kubectl delete secret nginxsecret -n %s --kubeconfig=%s", namespace, kubeconfig)
	util.ShellMuteOutput("kubectl delete policy -n %s default", namespace)
	util.ShellMuteOutput("kubectl delete destinationrule -n %s default", namespace)
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}


func Test16(t *testing.T) {
	log.Infof("# TC_16 Mutual TLS over HTTPS Services")
	// generate secrets
	// TBD 
	util.ShellSilent("openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/nginx.key -out /tmp/nginx.crt -subj \"/CN=my-nginx/O=my-nginx\"")
	util.CreateTLSSecret("nginxsecret", testNamespace, "/tmp/nginx.key", "/tmp/nginx.crt", kubeconfigFile)
	util.ShellSilent("kubectl create configmap nginxconfigmap --from-file=%s", nginxConf)
	
	t.Run("nginx_without_sidecar", func(t *testing.T) {
		log.Infof("Deploy an HTTPS service without the Istio sidecar")
		Inspect(deployNginx(false, testNamespace, kubeconfigFile), "failed to deploy nginx", "", t)
		Inspect(deploySleep(testNamespace, kubeconfigFile), "failed to deploy sleep", "", t)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
		Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://my-nginx -k | grep \"Welcome to nginx\"")
		msg, err := util.PodExec(testNamespace, sleepPod, "istio-proxy", cmd, true, kubeconfigFile)
		Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	util.KubeDelete(testNamespace, nginxNoSidecarYaml, kubeconfigFile)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)

	t.Run("nginx_with_sidecar", func(t *testing.T) {
		log.Info("Deploy an HTTPS service with the Istio sidecar and mutual TLS disabled")
		Inspect(deployNginx(true, testNamespace, kubeconfigFile), "failed to deploy nginx", "", t)
		Inspect(deploySleep(testNamespace, kubeconfigFile), "failed to deploy sleep", "", t)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
		Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://my-nginx -k | grep \"Welcome to nginx\"")
		msg, err := util.PodExec(testNamespace, sleepPod, "istio-proxy", cmd, true, kubeconfigFile)
		Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	util.KubeDelete(testNamespace, nginxYaml, kubeconfigFile)
	util.KubeDelete(testNamespace, sleepYaml, kubeconfigFile)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)

	t.Run("nginx_with_sidecar_mtls", func(t *testing.T) {
		log.Info("Enable mutual TLS")
		Inspect(util.KubeApplyContents(testNamespace, mtlsPolicy, kubeconfigFile), "failed to apply policy", "", t)
		mtlsRule := strings.Replace(mtlsRuleTemplate, "@token@", testNamespace, -1)
		Inspect(util.KubeApplyContents(testNamespace, mtlsRule, kubeconfigFile), "failed to apply rule", "", t)

		log.Info("Deploy an HTTPS service with Istio sidecar with mutual TLS enabled")
		Inspect(deploySleep(testNamespace, kubeconfigFile), "failed to deploy sleep", "", t)
		Inspect(deployNginx(true, testNamespace, kubeconfigFile), "failed to deploy nginx", "", t)

		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfigFile)
		Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://my-nginx -k | grep \"Welcome to nginx\"")
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", cmd, true, kubeconfigFile)
		Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
		
		msg, err = util.PodExec(testNamespace, sleepPod, "istio-proxy", cmd, true, kubeconfigFile)
		if err != nil {
			log.Infof("Expected fail from container istio-proxy: %v", err)
		} else {
			t.Errorf("Expected fail from container istio-proxy. Got unexpected response: %s", msg)
			log.Errorf("Expected fail from container istio-proxy. Got unexpected response: %s", msg)
		}
	})

	defer cleanup16(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test failed: %v", err)
		}
	}()
}
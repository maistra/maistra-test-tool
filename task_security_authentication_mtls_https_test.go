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

func cleanupAuthMTLSHTTPS(namespace string) {
	log.Info("# Cleanup ...")

	cleanNginx(namespace)
	cleanSleep(namespace)
	util.ShellMuteOutput("kubectl delete configmap nginxconfigmap -n %s", namespace)
	util.ShellMuteOutput("kubectl delete secret nginxsecret -n %s", namespace)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
}

func TestAuthMTLSHTTPS(t *testing.T) {
	defer cleanupAuthMTLSHTTPS(testNamespace)
	defer recoverPanic(t)

	log.Info("Mutual TLS over HTTPS")
	log.Info("Generate certificates and configmap")

	// generate secrets
	util.ShellMuteOutput("openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout /tmp/nginx.key -out /tmp/nginx.crt -subj \"/CN=my-nginx/O=my-nginx\"")
	util.CreateTLSSecret("nginxsecret", testNamespace, "/tmp/nginx.key", "/tmp/nginx.crt", kubeconfig)
	util.ShellMuteOutput("kubectl create configmap -n %s nginxconfigmap --from-file=%s", testNamespace, nginxDefaultConfig)

	deploySleep(testNamespace)
	sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("Security_authentication_https_service_without_sidecar", func(t *testing.T) {
		defer recoverPanic(t)

		deployNginx(false, testNamespace)

		cmd := fmt.Sprintf("curl https://my-nginx -k | grep \"Welcome to nginx\"")
		msg, err := util.PodExec(testNamespace, sleepPod, "istio-proxy", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	cleanNginx(testNamespace)

	t.Run("Security_authentication_https_service_with_sidecar", func(t *testing.T) {
		defer recoverPanic(t)

		deployNginx(true, testNamespace)

		cmd := fmt.Sprintf("curl https://my-nginx -k | grep \"Welcome to nginx\"")
		msg, err := util.PodExec(testNamespace, sleepPod, "istio-proxy", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	cleanNginx(testNamespace)
	cleanSleep(testNamespace)

	t.Run("Security_authentication_https_service_with_sidecar_with_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		// update mtls to true
		log.Info("Update SMCP mtls to true")
		util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*4) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

		deploySleep(testNamespace)
		sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		deployNginx(true, testNamespace)

		cmd := fmt.Sprintf("curl https://my-nginx -k | grep \"Welcome to nginx\"")
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "Welcome to nginx") {
			t.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
			log.Errorf("Expected Welcome to nginx; Got unexpected response: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		msg, err = util.PodExec(testNamespace, sleepPod, "istio-proxy", cmd, true, kubeconfig)
		if err != nil {
			log.Infof("Expected fail from container istio-proxy: %v", err)
		} else {
			t.Errorf("Expected fail from container istio-proxy. Got unexpected response: %s", msg)
			log.Errorf("Expected fail from container istio-proxy. Got unexpected response: %s", msg)
		}
	})
}

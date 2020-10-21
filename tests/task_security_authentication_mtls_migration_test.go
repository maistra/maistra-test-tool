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

package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupAuthMTLSMigration(namespace string) {
	log.Info("# Cleanup ...")

	util.KubeApplyContents(meshNamespace, PeerAuthPolicyPermissive, kubeconfig)
	util.KubeDeleteContents("foo", httpbinStrictPolicy, kubeconfig)

	namespaces := []string{"foo", "bar", "legacy"}
	for _, ns := range namespaces {
		util.KubeDelete(ns, sleepYaml, kubeconfig)
		util.KubeDelete(ns, httpbinYaml, kubeconfig)
	}
	util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`, meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)
}

func TestAuthMTLSMigration(t *testing.T) {
	defer cleanupAuthMTLSMigration(testNamespace)
	defer recoverPanic(t)

	log.Info("Mutual TLS Migration")

	util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`, meshNamespace, smcpName)
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(waitTime*2) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=ingressgateway", kubeconfig)
	util.CheckPodRunning(meshNamespace, "istio=egressgateway", kubeconfig)

	deployHttpbin("foo")
	deployHttpbin("bar")
	deploySleep("foo")
	deploySleep("bar")
	util.KubeApply("legacy", sleepLegacyYaml, kubeconfig)
	util.CheckPodRunning("legacy", "app=sleep", kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)

	log.Info("Verify setup")
	for _, from := range []string{"foo", "bar", "legacy"} {
		for _, to := range []string{"foo", "bar"} {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"", to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("Verify setup expected 200; Got unexpected response code: %s", msg)
				log.Errorf("Verify setup expected 200; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	log.Info("Check existing authentication policies or destination rules")
	util.Shell("kubectl get peerauthentication --all-namespaces | grep -v %s", meshNamespace)
	util.Shell("kubectl get destinationrule -n %s", meshNamespace)

	t.Run("Security_authentication_namespace_enable_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Lock down to mutual TLS by namespace")
		util.KubeApplyContents("foo", httpbinStrictPolicy, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		for _, from := range []string{"foo", "bar", "legacy"} {
			for _, to := range []string{"foo", "bar"} {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"", to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)

				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected sleep.legacy to httpbin.foo fails: %v", err)
					} else {
						t.Errorf("Expected sleep.legacy to httpbin.foo fails; Got unexpected response: %s", msg)
						log.Errorf("Expected sleep.legacy to httpbin.foo fails; Got unexpected response: %s", msg)
					}
					continue
				}

				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("MTLS traffic expected 200; Got unexpected response code: %s", msg)
					log.Errorf("MTLS traffic expected 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	util.KubeDeleteContents("foo", httpbinStrictPolicy, kubeconfig)

	t.Run("Security_authentication_globally_enable_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Lock down to mutual TLS for the entire cluster")
		log.Info("Globally enabling Istio mutual TLS")
		util.KubeApplyContents(meshNamespace, PeerAuthPolicyStrict, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		for _, from := range []string{"foo", "bar", "legacy"} {
			for _, to := range []string{"foo", "bar"} {

				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"", to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)

				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected sleep.legacy to httpbin.foo fails: %v", err)
					} else {
						t.Errorf("Expected sleep.legacy to httpbin.foo fails; Got unexpected response: %s", msg)
						log.Errorf("Expected sleep.legacy to httpbin.foo fails; Got unexpected response: %s", msg)
					}
					continue
				}

				if from == "legacy" && to == "bar" {
					if err != nil {
						log.Infof("Expected sleep.legacy to httpbin.bar fails: %v", err)
					} else {
						t.Errorf("Expected sleep.legacy to httpbin.bar fails; Got unexpected response: %s", msg)
						log.Errorf("Expected sleep.legacy to httpbin.bar fails; Got unexpected response: %s", msg)
					}
					continue
				}
			}
		}
	})
}

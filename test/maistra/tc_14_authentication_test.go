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


func cleanup14(kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.KubeDelete("foo", httpbinYaml, kubeconfig)
	util.KubeDelete("foo", sleepYaml, kubeconfig)
	util.KubeDelete("bar", httpbinYaml, kubeconfig)
	util.KubeDelete("bar", sleepYaml, kubeconfig)
	util.KubeDelete("legacy", httpbinLegacyYaml, kubeconfig)
	util.KubeDelete("legacy", sleepLegacyYaml, kubeconfig)
	util.ShellSilent("kubectl delete meshpolicy default")
	util.ShellSilent("kubectl delete destinationrules httpbin-legacy")
	util.ShellSilent("kubectl delete destinationrules -n default api-server")
	util.ShellSilent("kubectl delete destinationrule default -n default")
	util.ShellSilent("kubectl delete policy default overwrite-example -n foo")
	util.ShellSilent("kubectl delete policy httpbin -n bar")
	util.ShellSilent("kubectl delete destinationrules default overwrite-example -n foo")
	util.ShellSilent("kubectl delete destinationrules httpbin -n bar")
	util.ShellSilent("kubectl delete policy jwt-example -n foo")
	util.ShellSilent("kubectl delete policy httpbin -n bar")
	util.ShellSilent("kubectl delete destinationrule httpbin -n foo")
	util.ShellSilent("kubectl delete gateway httpbin-gateway -n foo")
	util.ShellSilent("kubectl delete virtualservice httpbin -n foo")
	util.DeleteNamespace("foo bar legacy", kubeconfig)
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}

func cleanupPart1() {
	log.Infof("# Cleanup part 1. Following error can be ignored...")
	util.ShellMuteOutput("kubectl delete meshpolicy default")
	util.ShellMuteOutput("kubectl delete destinationrules httpbin-legacy")
	util.ShellMuteOutput("kubectl delete destinationrules -n default api-server")
	util.ShellMuteOutput("kubectl delete destinationrule default -n default")
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func cleanupPart2() {
	log.Infof("# Cleanup part 2. Following error can be ignored...")
	util.ShellMuteOutput("kubectl delete policy default overwrite-example -n foo")
	util.ShellMuteOutput("kubectl delete policy httpbin -n bar")
	util.ShellMuteOutput("kubectl delete destinationrules default overwrite-example -n foo")
	util.ShellMuteOutput("kubectl delete destinationrules httpbin -n bar")
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func cleanupPart3() {
	log.Infof("# Cleanup part 3. Following error can be ignored...")
	util.ShellMuteOutput("kubectl delete policy jwt-example -n foo")
	util.ShellMuteOutput("kubectl delete policy httpbin -n bar")
	util.ShellMuteOutput("kubectl delete destinationrule httpbin -n foo")
	util.ShellMuteOutput("kubectl delete gateway httpbin-gateway -n foo")
	util.ShellMuteOutput("kubectl delete virtualservice httpbin -n foo")
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
}

func setup14(kubeconfig string) error {
	if err := util.KubeApply("foo", httpbinYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("foo", sleepYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("bar", httpbinYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("bar", sleepYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("legacy", httpbinLegacyYaml, kubeconfig); err != nil {
		return err
	}
	if err := util.KubeApply("legacy", sleepLegacyYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning("foo", "app=httpbin", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("foo", "app=sleep", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("bar", "app=httpbin", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("bar", "app=sleep", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("legacy", "app=httpbin", kubeconfigFile); err != nil {
		return err
	}
	if err := util.CheckPodRunning("legacy", "app=sleep", kubeconfigFile); err != nil {
		return err
	}
	log.Info("Waiting for rules to be cleaned up. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func checkPolicy() {
	util.Shell("kubectl get policies.authentication.istio.io --all-namespaces")
	util.Shell("kubectl get meshpolicies.authentication.istio.io")
	util.Shell("kubectl get destinationrules.networking.istio.io --all-namespaces -o yaml | grep \"host:\"")
}

// TBD convert pipes with Go strings methods
func getSecretToken() (string, error) {
	token, err := util.ShellMuteOutput("kubectl describe secret $(kubectl get secrets | grep default-token | cut -f1 -d ' ' | head -1) | grep -E '^token' | cut -f2 -d':' | tr -d '\\t'")
	if err != nil {
		return "", err
	}
	token = strings.Trim(token, "\t")
	token = strings.Trim(token, "\n")
	return token, nil
}

func jwcryptoInstall() {
	util.ShellSilent("sudo pip install jwcrypto")
	util.ShellSilent("chmod +x " + jwtGen)
}

func jwcryptoCleanup() {
	util.ShellSilent("sudo pip uninstall -y jwcrypto")
	util.ShellSilent("sudo pip uninstall -y pycparser")
	util.ShellSilent("sudo pip uninstall -y cffi")
	util.ShellSilent("sudo pip uninstall -y asn1crypto")
	util.ShellSilent("sudo pip uninstall -y cryptography")

}


func Test14(t *testing.T) {
	defer cleanup14(kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			log.Infof("Test panic: %v", err)
		}
	}()

	log.Infof("# TC_14 Authentication Policy")
	namespaces := []string{"foo", "bar", "legacy"}

	// Create namespaces
	for _, ns := range namespaces {
		Inspect(util.CreateNamespace(ns, kubeconfigFile), "failed to create namespace", "", t)
		OcGrantPermission("default", ns, kubeconfigFile)
	}

	Inspect(setup14(kubeconfigFile), "failed to apply deployments", "", t)
	log.Info("Verify setup")
	
	for _, from := range namespaces {
		for _, to := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
								to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("Verify setup -- Unexpected response code: %s", msg)
				log.Errorf("Verify setup -- Unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	t.Run("global_mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Globally enabling Istio mutual TLS")
		Inspect(util.KubeApplyContents("", meshPolicy, kubeconfigFile), "failed to apply MeshPolicy", "", t)
		log.Info("Waiting for rules to propagate. Sleep 60 seconds...")
		time.Sleep(time.Duration(60) * time.Second)

		ns := []string{"foo", "bar"}
		for _, from := range ns {
			for _, to := range ns {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
				Inspect(err, "failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
								to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
				Inspect(err, "failed to get response", "", t)
				if !strings.Contains(msg, "503") {
					t.Errorf("global mTLS expected: 503; Got response code: %s", msg)
					log.Errorf("global mTLS expected: 503; Got response code: %s", msg)
				} else {
					log.Infof("response 503 as expected: %s", msg)
				}
			}
		}

		Inspect(util.KubeApplyContents("", clientRule, kubeconfigFile), "failed to apply clientRule", "", t)
		log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
		time.Sleep(time.Duration(30) * time.Second)
		for _, from := range ns {
			for _, to := range ns {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
				Inspect(err, "failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
								to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
				Inspect(err, "failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("global mTLS expected: 200; Got response code: %s", msg)
					log.Errorf("global mTLS expected: 200; Got response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("non_istio_to_istio", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Request from non-Istio services to Istio services")

		ns := []string{"foo", "bar", "legacy"}
		from := "legacy"
		for _, to := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
								to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			if to != "legacy" && err != nil {
				continue
			} else if to != "legacy" && err == nil {
				t.Errorf("non Istio to Istio service expected: failed; Got unexpected response: %s", msg)
				log.Errorf("non Istio to Istio service expected: failed; Got unexpected response: %s", msg)
			} else if to == "legacy" && err != nil {
				t.Errorf("non Istio to Istio service legacy expected: not fail; Got unexpected response: %v", err)
				log.Errorf("non Istio to Istio service legacy expected: not fail; Got unexpected response: %v", err)
			} else {
				if !strings.Contains(msg, "200") {
					t.Errorf("non Istio to Istio service expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("non Istio to Istio service expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}	
		}
		log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
	})

	t.Run("istio_to_non_istio", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Request from Istio services to non-Istio services")
		ns := []string{"foo", "bar"}
		to := "legacy"
		for _, from := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
								to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "503") {
				t.Errorf("istio to non istio response expected: 503; Got unexpected response code: %s", msg)
				log.Errorf("istio to non istio response expected: 503; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Response 503 as expected : %s", msg)
			}
		}

		Inspect(util.KubeApplyContents("", legacyRule, kubeconfigFile), "failed to apply legacyRule", "", t)
		for _, from := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
			Inspect(err, "failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
								to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
			Inspect(err, "failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				t.Errorf("istio to non istio expected: 200; Got unexpected response code: %s", msg)
				log.Errorf("istio to non istio expected: 200; Got unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	})

	t.Run("istio_to_k8s_api", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Request from Istio services to Kubernetes API server")
		token, err := getSecretToken()
		Inspect(err, "failed to get secret token", "", t)
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfigFile)
		Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl https://kubernetes.default/api --header \"Authorization: Bearer %s\" --insecure -s -o /dev/null -w \"%%{http_code}\"", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		if err != nil {
			log.Infof("curl fails as expected: %v", err)
		} else {
			t.Errorf("istio to Kubernetes API server expected: failed; Got unexpected response: %s", msg)
			log.Errorf("istio to Kubernetes API server expected: failed; Got unexpected response: %s", msg)
		}

		Inspect(util.KubeApplyContents("", apiServerRule, kubeconfigFile), "failed to apply api-server rule", "", t)
		log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
		time.Sleep(time.Duration(20) * time.Second)

		msg, err = util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("istio to Kubernetes API server expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("istio to Kubernetes API server expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})

	cleanupPart1()

	t.Run("namespace_mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Enable mutual TLS per namespace")
		Inspect(util.KubeApplyContents("", fooPolicy, kubeconfigFile), "failed to apply foo Policy", "", t)
		Inspect(util.KubeApplyContents("", fooRule, kubeconfigFile), "failed to apply foo rule", "", t)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
				Inspect(err, "failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
									to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
				
				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
					continue
				}

				Inspect(err, "failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("namespace mTLS expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("namespace mTLS expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("service_mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Enable mutual TLS per service")
		Inspect(util.KubeApplyContents("", barPolicy, kubeconfigFile), "failed to apply bar Policy", "", t)
		Inspect(util.KubeApplyContents("", barRule, kubeconfigFile), "failed to apply bar rule", "", t)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
				Inspect(err, "failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
									to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)

				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
					continue
				}

				if from == "legacy" && to == "bar" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.bar: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
					}
					continue
				}

				Inspect(err, "failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("mTLS per service expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("mTLS per service expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("port_mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Edit mutual TLS only on httpbin bar port 1234")
		Inspect(util.KubeApplyContents("", barPolicy2, kubeconfigFile), "failed to apply bar Policy 2", "", t)
		Inspect(util.KubeApplyContents("", barRule2, kubeconfigFile), "failed to apply bar Rule 2", "", t)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
				Inspect(err, "failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
									to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
				
				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
					continue
				}

				Inspect(err, "failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("port mTLS expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("port mTLS expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("overwrite_policy", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()

		log.Info("Overwrite foo namespace policy by service policy")
		Inspect(util.KubeApplyContents("", fooPolicy2, kubeconfigFile), "failed to apply foo Policy 2", "", t)
		Inspect(util.KubeApplyContents("", fooRule2, kubeconfigFile), "failed to apply foo Rule 2", "", t)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfigFile)
				Inspect(err, "failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
									to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfigFile)
				Inspect(err, "failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	cleanupPart2()

	t.Run("end_user_auth", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("End-user authentication")
		ingress, err := GetOCPIngressgateway("app=istio-ingressgateway", "istio-system", kubeconfigFile)
		Inspect(err, "failed to get ingressgateway URL", "", t)
		url := fmt.Sprintf("http://%s/headers", ingress)

		Inspect(util.KubeApplyContents("", fooGateway, kubeconfigFile), "failed to apply foo gateway", "", t)
		Inspect(util.KubeApplyContents("", fooVS, kubeconfigFile), "failed to apply foo virtualservice", "", t)
		log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
		time.Sleep(time.Duration(10) * time.Second)
		
		resp, _, err := GetHTTPResponse(url, nil)
		Inspect(err, "failed to get httpbin header response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get response: %d", resp.StatusCode)
		}
		CloseResponseBody(resp)

		util.Shell("kubectl get policies.authentication.istio.io -n foo")

		Inspect(util.KubeApplyContents("foo", fooJWTPolicy, kubeconfigFile), "failed to apply foo JWT Policy", "", t)
		log.Info("Waiting for rules to propagate. Sleep 45 seconds...")
		time.Sleep(time.Duration(45) * time.Second)

		resp, _, err = GetHTTPResponse(url, nil)
		CloseResponseBody(resp)
		resp, _, err = GetHTTPResponse(url, nil)
		CloseResponseBody(resp)
		resp, _, err = GetHTTPResponse(url, nil)
		Inspect(err, "failed to get httpbin header response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Expected: 401; Got unexpected response code: %d %s", resp.StatusCode, resp.Status)
			log.Errorf("Expected: 401; Got unexpected response code: %d %s", resp.StatusCode, resp.Status)
		} else {
			log.Infof("Success. Get expected response 401: %d", resp.StatusCode)
		}
		CloseResponseBody(resp)

		log.Info("Attaching the valid token")
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		Inspect(err, "failed to get JWT token", "", t)
		resp, err = GetWithJWT(url, token, "")
		Inspect(err, "failed to get httpbin header response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get response: %d", resp.StatusCode)
		}
		CloseResponseBody(resp)
		
		log.Info("Test JWT expires in 5 seconds")
		jwcryptoInstall()
		log.Info("Check curls return five or six 200 and then five or four 401")
		token, err = util.ShellSilent("%s %s --expire 5", jwtGen, jwtKey)
		token = strings.Trim(token, "\n")
		Inspect(err, "failed to get JWT token", "", t)
		for i := 0; i < 10; i++ {
			resp, err = GetWithJWT(url, token, "")
			Inspect(err, "failed to get httpbin header response", "", t)
			if i == 0 {
				if resp.StatusCode != 200 {
					t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
					log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
				} else {
					log.Infof("Success. Get response: %d", resp.StatusCode)
				}
			} 
			if i > 5 {
				if resp.StatusCode != 401 {
					t.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
					log.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
				} else {
					log.Infof("Success. Get expected response 401: %d", resp.StatusCode)
				}
			}
			CloseResponseBody(resp)
			time.Sleep(time.Duration(1) * time.Second)
		}
		jwcryptoCleanup()
	})

	t.Run("end_user_auth_mTLS", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				log.Infof("Test panic: %v", err)
			}
		}()
		
		log.Info("End-user authentication with mutual TLS")
		Inspect(util.KubeApplyContents("", fooJWTPolicy2, kubeconfigFile), "failed to apply foo JWT Policy 2", "", t)		
		Inspect(util.KubeApplyContents("", fooRule3, kubeconfigFile), "failed to apply foo rule 3", "", t)
		
		log.Info("Check request from istio services")
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		Inspect(err, "failed to get JWT token", "", t)
		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfigFile)
		Inspect(err, "failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfigFile)
		Inspect(err, "failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Check request from non-istio services")
		sleepPod, err = util.GetPodName("legacy", "app=sleep", kubeconfigFile)
		Inspect(err, "failed to get sleep pod name", "", t)
		msg, err = util.PodExec("legacy", sleepPod, "sleep", cmd, true, kubeconfigFile)
		if err != nil {
			log.Infof("Expected failed request from non-istio services: %v", err)
		} else {
			t.Errorf("Expecte failed request; Got unexpected response: %s", msg)
			log.Errorf("Expecte failed request; Got unexpected response: %s", msg)
		}
	})
	
	cleanupPart3()

}
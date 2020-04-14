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

	"istio.io/istio/pkg/log"
)

func cleanupAuthPolicy() {
	log.Info("# Cleanup ...")

	util.KubeDeleteContents("foo", fooMTLSRule, kubeconfig)
	util.Shell("rm -f gen-jwt.py")
	util.Shell("rm -f key.pem")
	util.KubeDeleteContents("foo", fooJWTPolicy, kubeconfig)
	util.KubeDeleteContents("foo", fooGateway, kubeconfig)
	util.KubeDeleteContents("foo", fooVS, kubeconfig)

	util.KubeDeleteContents("foo", fooPolicyOverwrite, kubeconfig)
	util.KubeDeleteContents("foo", fooRuleOverwrite, kubeconfig)
	util.KubeDeleteContents("bar", barPortPolicy, kubeconfig)
	util.KubeDeleteContents("bar", barPortRule, kubeconfig)
	util.KubeDeleteContents("bar", barPolicy, kubeconfig)
	util.KubeDeleteContents("bar", barRule, kubeconfig)
	util.KubeDeleteContents("foo", fooPolicy, kubeconfig)
	util.KubeDeleteContents("foo", fooRule, kubeconfig)

	util.KubeDeleteContents("legacy", legacyRule, kubeconfig)
	util.KubeDeleteContents(meshNamespace, clientRule, kubeconfig)
	util.Shell("kubectl patch -n %s servicemeshpolicy/%s --type merge -p '{\"spec\":{\"peers\":[{\"mtls\":{\"mode\": \"PERMISSIVE\"}}]}}'", meshNamespace, "default")

	namespaces := []string{"foo", "bar", "legacy"}
	for _, ns := range namespaces {
		util.KubeDelete(ns, sleepYaml, kubeconfig)
		util.KubeDelete(ns, httpbinYaml, kubeconfig)
	}
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func TestAuthPolicy(t *testing.T) {
	defer cleanupAuthPolicy()
	defer recoverPanic(t)

	log.Infof("# Authentication Policy")
	// setup
	namespaces := []string{"foo", "bar", "legacy"}

	deployHttpbin("foo")
	deployHttpbin("bar")
	deploySleep("foo")
	deploySleep("bar")
	util.KubeApply("legacy", httpbinLegacyYaml, kubeconfig)
	util.CheckPodRunning("legacy", "app=httpbin", kubeconfig)
	util.KubeApply("legacy", sleepLegacyYaml, kubeconfig)
	util.CheckPodRunning("legacy", "app=sleep", kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)

	log.Info("Verify setup")
	for _, from := range namespaces {
		for _, to := range namespaces {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				log.Errorf("Verify setup -- Unexpected response code: %s", msg)
			} else {
				log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	t.Run("Security_authentication_enable_global_mTLS", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Globally enabling Istio mutual TLS")
		util.Shell("kubectl patch -n %s servicemeshpolicy/%s --type merge -p '{\"spec\":{\"peers\":[{\"mtls\":{}}]}}'", meshNamespace, "default")
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		ns := []string{"foo", "bar"}
		for _, from := range ns {
			for _, to := range ns {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "503") {
					t.Errorf("Global mTLS expected: 503; Got response code: %s", msg)
					log.Errorf("Global mTLS expected: 503; Got response code: %s", msg)
				} else {
					log.Infof("Response 503 as expected: %s", msg)
				}
			}
		}

		util.KubeApplyContents(meshNamespace, clientRule, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
		time.Sleep(time.Duration(waitTime*6) * time.Second)
		for _, from := range ns {
			for _, to := range ns {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("Global mTLS expected: 200; Got response code: %s", msg)
					log.Errorf("Global mTLS expected: 200; Got response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("Security_authentication_request_non-istio_to_istio_services", func(t *testing.T) {
		defer recoverPanic(t)

		ns := []string{"foo", "bar"}
		from := "legacy"
		for _, to := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			if err != nil {
				log.Infof("Response failed as expected: %s", msg)
			} else {
				t.Errorf("Unexpected request from non-istio to istio services: %s", msg)
				log.Errorf("Unexpected request from non-istio to istio services: %s", msg)
			}
		}
	})

	t.Run("Security_authentication_request_istio_to_non-istio_services", func(t *testing.T) {
		defer recoverPanic(t)

		ns := []string{"foo", "bar"}
		to := "legacy"
		for _, from := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			if !strings.Contains(msg, "503") {
				t.Errorf("Request from istio to non-istio expected: 503; Got response code: %s", msg)
				log.Errorf("Request from istio to non-istio expected: 503; Got response code: %s", msg)
			} else {
				log.Infof("Response 503 as expected: %s", msg)
			}
		}

		log.Info("Add a destination rule for httpbin.legacy")
		util.KubeApplyContents("legacy", legacyRule, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		for _, from := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			if !strings.Contains(msg, "200") {
				t.Errorf("Request from istio to non-istio expected: 200; Got response code: %s", msg)
				log.Errorf("Request from istio to non-istio expected: 200; Got response code: %s", msg)
			} else {
				log.Infof("Response 200 as expected: %s", msg)
			}
		}
	})

	// istio_to_k8s_api_test

	// cleanup part 1
	util.KubeDeleteContents("legacy", legacyRule, kubeconfig)
	util.KubeDeleteContents(meshNamespace, clientRule, kubeconfig)
	util.Shell("kubectl patch -n %s servicemeshpolicy/%s --type merge -p '{\"spec\":{\"peers\":[{\"mtls\":{\"mode\": \"PERMISSIVE\"}}]}}'", meshNamespace, "default")
	log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
	time.Sleep(time.Duration(waitTime*10) * time.Second)

	t.Run("Security_authentication_namespace_policy_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Enable mutual TLS per namespace")
		util.KubeApplyContents("foo", fooPolicy, kubeconfig)
		util.KubeApplyContents("foo", fooRule, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)

				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
					continue
				}

				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					log.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
					t.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
		util.KubeDeleteContents("foo", fooPolicy, kubeconfig)
		util.KubeDeleteContents("foo", fooRule, kubeconfig)
	})

	t.Run("Security_authentication_service_policy_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Enable mutual TLS per service")
		util.KubeApplyContents("bar", barPolicy, kubeconfig)
		util.KubeApplyContents("bar", barRule, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)

				if from == "legacy" && to == "bar" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.bar: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
					}
					continue
				}

				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("MTLS per service expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("MTLS per service expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("Security_authentication_port_policy_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Edit mutual TLS only on httpbin bar port 1234")
		util.KubeApplyContents("bar", barPortPolicy, kubeconfig)
		util.KubeApplyContents("bar", barPortRule, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("Port MTLS expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("Port MTLS expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	t.Run("Security_authentication_policy_precedence_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Overwrite foo namespace policy by service policy")
		util.KubeApplyContents("foo", fooPolicy, kubeconfig)
		util.KubeApplyContents("foo", fooRule, kubeconfig)
		util.KubeApplyContents("foo", fooPolicyOverwrite, kubeconfig)
		util.KubeApplyContents("foo", fooRuleOverwrite, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf("curl http://httpbin.%s:8000/ip -s -o /dev/null -w \"sleep.%s to httpbin.%s: %%{http_code}\"",
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
				util.Inspect(err, "Failed to get response", "", t)
				if !strings.Contains(msg, "200") {
					t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
					log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
				} else {
					log.Infof("Success. Get expected response: %s", msg)
				}
			}
		}
	})

	// cleanup part 2
	util.KubeDeleteContents("foo", fooPolicyOverwrite, kubeconfig)
	util.KubeDeleteContents("foo", fooRuleOverwrite, kubeconfig)
	util.KubeDeleteContents("bar", barPortPolicy, kubeconfig)
	util.KubeDeleteContents("bar", barPortRule, kubeconfig)
	util.KubeDeleteContents("bar", barPolicy, kubeconfig)
	util.KubeDeleteContents("bar", barRule, kubeconfig)
	util.KubeDeleteContents("foo", fooPolicy, kubeconfig)
	util.KubeDeleteContents("foo", fooRule, kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)

	t.Run("Security_authentication_end-user_JWT", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("End-user authentication")
		url := fmt.Sprintf("http://%s/headers", gatewayHTTP)

		util.KubeApplyContents("foo", fooGateway, kubeconfig)
		util.KubeApplyContents("foo", fooVS, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		resp, _, err := util.GetHTTPResponse(url, nil)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		log.Info("check existing policy in foo")
		util.Shell("kubectl get policies.authentication.istio.io -n foo")

		util.KubeApplyContents("foo", fooJWTPolicy, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		resp, _, err = util.GetHTTPResponse(url, nil)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Expected: 401; Got unexpected response code: %d %s", resp.StatusCode, resp.Status)
			log.Errorf("Expected: 401; Got unexpected response code: %d %s", resp.StatusCode, resp.Status)
		} else {
			log.Infof("Success. Get expected response 401: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		log.Info("Attaching the valid token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.4/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		resp, err = util.GetWithJWT(url, token, "")
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		log.Info("Test JWT expires in 5 seconds")
		jwtGen := "https://raw.githubusercontent.com/istio/istio/release-1.4/security/tools/jwt/samples/gen-jwt.py"
		jwtKey := "https://raw.githubusercontent.com/istio/istio/release-1.4/security/tools/jwt/samples/key.pem"

		util.ShellMuteOutput("pip install jwcrypto")
		util.Shell("wget %s", jwtGen)
		util.Shell("chmod +x gen-jwt.py")
		util.Shell("wget %s", jwtKey)

		log.Info("Check curls return severl 200s and then severl 401s")
		token, err = util.ShellMuteOutput("python gen-jwt.py key.pem --expire 5")
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)
		resp, err = util.GetWithJWT(url, token, "")
		log.Infof("Success. Get response: %d", resp.StatusCode)

		time.Sleep(time.Duration(7) * time.Second)
		resp, err = util.GetWithJWT(url, token, "")
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get expected response 401: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("Security_authentication_end-user_per-path_JWT", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("End-user authentication with per-path requirements")

		log.Info("Disable End-user authentication for specific paths")
		util.KubeApplyContents("foo", fooJWTUserAgentPolicy, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		url := fmt.Sprintf("http://%s/user-agent", gatewayHTTP)
		resp, _, err := util.GetHTTPResponse(url, nil)
		util.Inspect(err, "Failed to get httpbin user-agent response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get httpbin user-agent response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		url = fmt.Sprintf("http://%s/headers", gatewayHTTP)
		resp, _, err = util.GetHTTPResponse(url, nil)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get httpbin header response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		log.Info("Enable End-user authentication for specific paths")
		util.KubeApplyContents("foo", fooJWTIPPolicy, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		url = fmt.Sprintf("http://%s/user-agent", gatewayHTTP)
		resp, _, err = util.GetHTTPResponse(url, nil)
		util.Inspect(err, "Failed to get httpbin user-agent response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get httpbin user-agent response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		url = fmt.Sprintf("http://%s/ip", gatewayHTTP)
		resp, _, err = util.GetHTTPResponse(url, nil)
		util.Inspect(err, "Failed to get httpbin ip response", "", t)
		if resp.StatusCode != 401 {
			t.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 401; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get httpbin ip response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)

		log.Info("Attaching the valid token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.4/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		resp, err = util.GetWithJWT(url, token, "")
		util.Inspect(err, "Failed to get httpbin ip response", "", t)
		if resp.StatusCode != 200 {
			t.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
			log.Errorf("Expected: 200; Got unexpected response code: %d", resp.StatusCode)
		} else {
			log.Infof("Success. Get httpbin ip response: %d", resp.StatusCode)
		}
		util.CloseResponseBody(resp)
	})

	t.Run("Security_authentication_end-user_MTLS_JWT", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("End-user authentication with mutual TLS")
		util.KubeApplyContents("foo", fooJWTMTLSPolicy, kubeconfig)
		util.KubeApplyContents("foo", fooMTLSRule, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		log.Info("Check request from istio services")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.4/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf("curl http://httpbin.foo:8000/ip -s -o /dev/null -w \"%%{http_code}\" --header \"Authorization: Bearer %s\"", token)
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}

		log.Info("Check request from non-istio services")
		sleepPod, err = util.GetPodName("legacy", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		msg, err = util.PodExec("legacy", sleepPod, "sleep", cmd, true, kubeconfig)
		if err != nil {
			log.Infof("Expected failed request from non-istio services: %v", err)
		} else {
			t.Errorf("Expecte failed request; Got unexpected response: %s", msg)
			log.Errorf("Expecte failed request; Got unexpected response: %s", msg)
		}
	})

}

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

package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupAuthPolicy() {
	log.Info("# Cleanup ...")

	util.KubeDeleteContents("foo", fooMTLSRule, kubeconfig)
	util.Shell("rm -f gen-jwt.py")
	util.Shell("rm -f key.pem")
	util.KubeDeleteContents(meshNamespace, fooJWTPathPolicy, kubeconfig)
	util.KubeDeleteContents(meshNamespace, fooJWTPolicy, kubeconfig)
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

	util.KubeApplyContents(meshNamespace, PeerAuthPolicyPermissive, kubeconfig)

	namespaces := []string{"foo", "bar", "legacy"}
	for _, ns := range namespaces {
		util.KubeDelete(ns, sleepYaml, kubeconfig)
		util.KubeDelete(ns, httpbinYaml, kubeconfig)
	}
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestAuthPolicy(t *testing.T) {
	defer cleanupAuthPolicy()
	defer recoverPanic(t)

	log.Infof("# Authentication Policy")
	// setup
	namespaces := []string{"foo", "bar"}

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
			cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
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

	t.Run("Security_authentication_auto_mTLS", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Auto mutual TLS")
		out, _ := util.Shell("kubectl exec $(kubectl get pod -l app=sleep -n foo -o jsonpath={.items..metadata.name}) -c sleep -n foo -- curl http://httpbin.foo:8000/headers -s | grep X-Forwarded-Client-Cert")
		if !strings.Contains(out, "X-Forwarded-Client-Cert") {
			t.Errorf("Auto mTLS failed to get X-Forwarded-Client-Cert")
			log.Info("Auto mTLS failed to get X-Forwarded-Client-Cert")
		}

		out, _ = util.ShellMuteOutput("kubectl exec $(kubectl get pod -l app=sleep -n foo -o jsonpath={.items..metadata.name}) -c sleep -n foo -- curl http://httpbin.legacy:8000/headers -s | grep X-Forwarded-Client-Cert")
		if strings.Contains(out, "X-Forwarded-Client-Cert") {
			t.Errorf("Auto mTLS legacy should not get X-Forwarded-Client-Cert")
			log.Info("Auto mTLS legacy should not to get X-Forwarded-Client-Cert")
		}
	})

	t.Run("Security_authentication_enable_global_mTLS", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Globally enabling Istio mutual TLS")
		util.KubeApplyContents(meshNamespace, PeerAuthPolicyStrict, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
		time.Sleep(time.Duration(waitTime*10) * time.Second)

		from := "legacy"
		ns := []string{"foo", "bar"}

		for _, to := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)
			if strings.Contains(msg, "200") {
				t.Errorf("Global mTLS expected 000; Got response code: %s", msg)
				log.Errorf("Global mTLS expected: 000; Got response code: %s", msg)
			} else {
				log.Infof("Response 000 as expected: %s", msg)
			}
		}
	})

	// cleanup part 1
	util.KubeApplyContents(meshNamespace, PeerAuthPolicyPermissive, kubeconfig)
	log.Info("Waiting for rules to propagate. Sleep 50 seconds...")
	time.Sleep(time.Duration(waitTime*10) * time.Second)

	t.Run("Security_authentication_namespace_policy_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Enable mutual TLS per namespace")
		util.KubeApplyContents("foo", fooPolicy, kubeconfig)
		//util.KubeApplyContents("foo", fooRule, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		namespaces := []string{"foo", "bar", "legacy"}
		for _, from := range namespaces {
			for _, to := range namespaces {
				sleepPod, err := util.GetPodName(from, "app=sleep", kubeconfig)
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true, kubeconfig)

				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
				} else {
					if !strings.Contains(msg, "200") {
						log.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
						t.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
					} else {
						log.Infof("Success. Get expected response: %s", msg)
					}
				}
			}
		}
		util.KubeDeleteContents("foo", fooPolicy, kubeconfig)
	})

	t.Run("Security_authentication_workload_policy_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Enable mutual TLS per workload")
		util.KubeApplyContents("bar", barPolicy, kubeconfig)
		util.KubeApplyContents("bar", barRule, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err := util.GetPodName("legacy", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"bar", "legacy", "bar")
		msg, err := util.PodExec("legacy", sleepPod, "sleep", cmd, true, kubeconfig)
		if err != nil {
			log.Infof("Expected fail from sleep.legacy to httpbin.bar: %v", err)
		} else {
			t.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
			log.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
		}
	})

	t.Run("Security_authentication_port_policy_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Edit mutual TLS only on httpbin bar port 1234")
		util.KubeApplyContents("bar", barPortPolicy, kubeconfig)
		util.KubeApplyContents("bar", barPortRule, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err := util.GetPodName("legacy", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"bar", "legacy", "bar")
		msg, err := util.PodExec("legacy", sleepPod, "sleep", cmd, true, kubeconfig)
		if strings.Contains(msg, "200") {
			log.Infof("Expected 200 from sleep.legacy to httpbin.bar: %s", msg)
		} else {
			t.Errorf("Expected 200 from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
			log.Errorf("Expected 200 from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
		}
	})

	t.Run("Security_authentication_policy_precedence_mtls", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Overwrite foo namespace policy by service policy")
		util.KubeApplyContents("foo", fooPolicy, kubeconfig)
		util.KubeApplyContents("foo", fooRule, kubeconfig)
		util.KubeApplyContents("foo", fooPolicyOverwrite, kubeconfig)
		util.KubeApplyContents("foo", fooRuleOverwrite, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err := util.GetPodName("legacy", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"foo", "legacy", "foo")
		msg, err := util.PodExec("legacy", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
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
	time.Sleep(time.Duration(waitTime*2) * time.Second)

	t.Run("Security_authentication_end-user_JWT", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("End-user authentication")
		util.KubeApplyContents("foo", fooGateway, kubeconfig)
		util.KubeApplyContents("foo", fooVS, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		msg, err := util.Shell(`curl %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get response: %s", msg)
		}

		util.KubeApplyContents(meshNamespace, fooJWTPolicy, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		msg, err = util.Shell(`curl --header "Authorization: Bearer deadbeef" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "401") {
			t.Errorf("Expected: 401; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 401; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response 401: %s", msg)
		}

		log.Info("Attaching the valid token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get response: %s", msg)
		}

		log.Info("Test JWT expires in 5 seconds")
		jwtGen := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/gen-jwt.py"
		jwtKey := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/key.pem"

		log.Info("Install python package jwcrypto using /usr/bin/python")
		util.Shell("/usr/bin/python -m pip install --user jwcrypto")
		util.Shell("curl -Lo ./gen-jwt.py %s", jwtGen)
		util.Shell("chmod +x gen-jwt.py")
		util.Shell("curl -Lo ./key.pem %s", jwtKey)

		log.Info("Check curls return severl 200s and then severl 401s")
		token, err = util.ShellMuteOutput("/usr/bin/python gen-jwt.py key.pem --expire 5")
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
                log.Infof("Success. Get response: %s", msg)
                log.Info("Added Wait 60 secs for clock sync issue")
                time.Sleep(time.Duration(60) * time.Second)

                msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
                log.Infof("Success. Get response: %s", msg)
                log.Info("Wait 10 secs")
                time.Sleep(time.Duration(10) * time.Second)

		//msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
		//log.Infof("Success. Get response: %s", msg)

		//time.Sleep(time.Duration(7) * time.Second)
		msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "401") {
			t.Errorf("Expected: 401; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 401; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response 401: %s", msg)
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("Security_authentication_end-user_per-path_JWT", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("End-user authentication with per-path requirements")

		log.Info("Disable End-user authentication for specific paths")
		util.KubeApplyContents(meshNamespace, fooJWTPathPolicy, kubeconfig)
		log.Info("Waiting for rules to propagate. Sleep 20 seconds...")
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		msg, err := util.Shell(`curl %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "403") {
			t.Errorf("Expected: 403; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 403; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get httpbin header response: %s", msg)
		}

		log.Info("Attaching the valid token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.6/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/ip -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin ip response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get httpbin header response: %s", msg)
		}
	})
}

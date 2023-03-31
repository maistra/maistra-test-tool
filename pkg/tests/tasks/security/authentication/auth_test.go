// Copyright 2021 Red Hat, Inc.
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

package authentication

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupAuthPolicy() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(RequireTokenPathPolicyTemplate, smcp))
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(RequireTokenPolicyTemplate, smcp))
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(JWTAuthPolicyTemplate, smcp))
	util.KubeDeleteContents("foo", HttpbinGateway)
	util.KubeDeleteContents("foo", OverwritePolicy)
	util.KubeDeleteContents("bar", PortPolicy)
	util.KubeDeleteContents("bar", WorkloadPolicyStrict)
	util.KubeDeleteContents("foo", NamespacePolicyStrict)
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(PeerAuthPolicyStrictTemplate, smcp))

	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	sleep = examples.Sleep{"bar"}
	httpbin = examples.Httpbin{"bar"}
	sleep.Uninstall()
	httpbin.Uninstall()
	sleep = examples.Sleep{"legacy"}
	httpbin = examples.Httpbin{"legacy"}
	sleep.Uninstall()
	httpbin.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestAuthPolicy(t *testing.T) {
	test.NewTest(t).Id("T18").Groups(test.Full, test.InterOp).NotRefactoredYet()

	defer cleanupAuthPolicy()
	defer util.RecoverPanic(t)

	log.Log.Info("Test Authentication Policy")
	httpbin := examples.Httpbin{"foo"}
	httpbin.Install()
	httpbin = examples.Httpbin{"bar"}
	httpbin.Install()
	httpbin = examples.Httpbin{"legacy"}
	httpbin.InstallLegacy()

	sleep := examples.Sleep{"foo"}
	sleep.Install()
	sleep = examples.Sleep{"bar"}
	sleep.Install()
	sleep = examples.Sleep{"legacy"}
	sleep.InstallLegacy()

	log.Log.Info("Verify setup")
	for _, from := range []string{"foo", "bar", "legacy"} {
		for _, to := range []string{"foo", "bar"} {
			sleepPod, err := util.GetPodName(from, "app=sleep")
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				log.Log.Errorf("Verify setup -- Unexpected response code: %s", msg)
			} else {
				log.Log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	log.Log.Info("Verify peer authentication policy")
	util.Shell(`kubectl get peerauthentication --all-namespaces`)
	log.Log.Info("Verify destination rules")
	util.Shell(`kubectl get destinationrules.networking.istio.io --all-namespaces -o yaml | grep "host:"`)

	t.Run("Security_authentication_auto_mTLS", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Auto mutual TLS")
		out, _ := util.Shell(`kubectl exec $(kubectl get pod -l app=sleep -n foo -o jsonpath={.items..metadata.name}) -c sleep -n foo -- curl http://httpbin.foo:8000/headers -s | grep X-Forwarded-Client-Cert | sed 's/Hash=[a-z0-9]*;/Hash=<redacted>;/'`)
		if !strings.Contains(out, "X-Forwarded-Client-Cert") {
			t.Errorf("Auto mTLS failed to get X-Forwarded-Client-Cert")
			log.Log.Info("Auto mTLS failed to get X-Forwarded-Client-Cert")
		}

		out, _ = util.ShellMuteOutputError(`kubectl exec $(kubectl get pod -l app=sleep -n foo -o jsonpath={.items..metadata.name}) -c sleep -n foo -- curl http://httpbin.legacy:8000/headers -s | grep X-Forwarded-Client-Cert`)
		if strings.Contains(out, "X-Forwarded-Client-Cert") {
			t.Errorf("Auto mTLS legacy should not get X-Forwarded-Client-Cert")
			log.Log.Info("Auto mTLS legacy should not to get X-Forwarded-Client-Cert")
		}
	})

	t.Run("Security_authentication_enable_global_mTLS_STRICT_mode", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Globally enabling Istio mutual TLS in STRICT mode")
		util.KubeApplyContents(meshNamespace, util.RunTemplate(PeerAuthPolicyStrictTemplate, smcp))
		log.Log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
		time.Sleep(time.Duration(30) * time.Second)

		from := "legacy"
		ns := []string{"foo", "bar"}
		for _, to := range ns {
			sleepPod, err := util.GetPodName(from, "app=sleep")
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true)
			if strings.Contains(msg, "200") {
				t.Errorf("Global mTLS expected 000; Got response code: %s", msg)
				log.Log.Errorf("Global mTLS expected: 000; Got response code: %s", msg)
			} else {
				log.Log.Infof("Response 000 as expected: %s", msg)
			}
		}
		util.KubeDeleteContents(meshNamespace, util.RunTemplate(PeerAuthPolicyStrictTemplate, smcp))
		time.Sleep(time.Duration(30) * time.Second)
	})

	t.Run("Security_authentication_namespace_policy_mtls", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Enable mutual TLS per namespace")
		util.KubeApplyContents("foo", NamespacePolicyStrict)
		time.Sleep(time.Duration(10) * time.Second)

		for _, from := range []string{"foo", "bar", "legacy"} {
			for _, to := range []string{"foo", "bar"} {
				sleepPod, err := util.GetPodName(from, "app=sleep")
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true)

				if from == "legacy" && to == "foo" {
					if err != nil {
						log.Log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						log.Log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
				} else {
					if !strings.Contains(msg, "200") {
						log.Log.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
						t.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
					} else {
						log.Log.Infof("Success. Get expected response: %s", msg)
					}
				}
			}
		}
		util.KubeDeleteContents("foo", NamespacePolicyStrict)
	})

	t.Run("Security_authentication_workload_policy_mtls", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Enable mutual TLS per workload")
		util.KubeApplyContents("bar", WorkloadPolicyStrict)
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err := util.GetPodName("legacy", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"bar", "legacy", "bar")
		msg, err := util.PodExec("legacy", sleepPod, "sleep", cmd, true)
		if err != nil {
			log.Log.Infof("Expected fail from sleep.legacy to httpbin.bar: %v", err)
		} else {
			t.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
			log.Log.Errorf("Expected fail from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
		}

		log.Log.Info("Refine mutual TLS per port")
		util.KubeApplyContents("bar", PortPolicy)
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err = util.GetPodName("legacy", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd = fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"bar", "legacy", "bar")
		msg, err = util.PodExec("legacy", sleepPod, "sleep", cmd, true)
		if strings.Contains(msg, "200") {
			log.Log.Infof("Expected 200 from sleep.legacy to httpbin.bar: %s", msg)
		} else {
			t.Errorf("Expected 200 from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
			log.Log.Errorf("Expected 200 from sleep.legacy to httpbin.bar; Got unexpected response: %s", msg)
		}
		util.KubeDeleteContents("bar", PortPolicy)
		util.KubeDeleteContents("bar", WorkloadPolicyStrict)
	})

	t.Run("Security_authentication_policy_precedence_mtls", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Overwrite foo namespace policy by a workload policy")
		util.KubeApplyContents("foo", OverwritePolicy)
		time.Sleep(time.Duration(10) * time.Second)

		sleepPod, err := util.GetPodName("legacy", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"foo", "legacy", "foo")
		msg, err := util.PodExec("legacy", sleepPod, "sleep", cmd, true)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response: %s", msg)
		}
		util.KubeDeleteContents("foo", OverwritePolicy)
	})

	t.Run("Security_authentication_end-user_JWT", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("End-user authentication")
		log.Log.Info("Apply httpbin gateway")
		util.KubeApplyContents("foo", HttpbinGateway)
		time.Sleep(time.Duration(20) * time.Second)

		msg, err := util.Shell(`curl %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get response: %s", msg)
		}

		log.Log.Info("Apply a JWT policy")
		util.KubeApplyContents(meshNamespace, util.RunTemplate(JWTAuthPolicyTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		log.Log.Info("Request without token returns 200. Request with an invalid token returns 401")
		msg, err = util.Shell(`curl %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get response: %s", msg)
		}

		msg, err = util.Shell(`curl --header "Authorization: Bearer deadbeef" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "401") {
			t.Errorf("Expected: 401; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 401; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get expected response 401: %s", msg)
		}

		log.Log.Info("Attaching the valid token")
		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt"
		token, err := util.ShellSilent("curl %s -s", jwtURL)
		token = strings.Trim(token, "\n")
		util.Inspect(err, "Failed to get JWT token", "", t)

		msg, err = util.Shell(`curl --header "Authorization: Bearer %s" %s/headers -s -o /dev/null -w "%%{http_code}\n"`, token, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get response: %s", msg)
		}

		// skip gen-jwt.py and test JWT expires
	})

	t.Run("Security_authentication_end-user_require_JWT", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Require a valid token")
		util.KubeApplyContents(meshNamespace, util.RunTemplate(RequireTokenPolicyTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		msg, err := util.Shell(`curl %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "403") {
			t.Errorf("Expected: 403; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 403; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get httpbin header response: %s", msg)
		}

		log.Log.Info("Require valid tokens per-path")
		util.KubeApplyContents(meshNamespace, util.RunTemplate(RequireTokenPathPolicyTemplate, smcp))
		time.Sleep(time.Duration(20) * time.Second)

		msg, err = util.Shell(`curl %s/headers -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin header response", "", t)
		if !strings.Contains(msg, "403") {
			t.Errorf("Expected: 403; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 403; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get httpbin header response: %s", msg)
		}

		msg, err = util.Shell(`curl %s/ip -s -o /dev/null -w "%%{http_code}\n"`, gatewayHTTP)
		util.Inspect(err, "Failed to get httpbin ip response", "", t)
		if !strings.Contains(msg, "200") {
			t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
			log.Log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		} else {
			log.Log.Infof("Success. Get httpbin header response: %s", msg)
		}
	})
}

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
)

func cleanupMigration() {
	util.Log.Info("Cleanup")
	util.KubeDeleteContents("istio-system", MeshPolicyStrict)
	util.KubeDeleteContents("foo", NamespacePolicyStrict)
	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	sleep = examples.Sleep{"bar"}
	httpbin = examples.Httpbin{"bar"}
	sleep.Uninstall()
	httpbin.Uninstall()
	sleep = examples.Sleep{"legacy"}
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestMigration(t *testing.T) {
	defer cleanupMigration()
	defer util.RecoverPanic(t)

	util.Log.Info("Mutual TLS Migration")
	httpbin := examples.Httpbin{"foo"}
	httpbin.Install()
	httpbin = examples.Httpbin{"bar"}
	httpbin.Install()
	sleep := examples.Sleep{"foo"}
	sleep.Install()
	sleep = examples.Sleep{"bar"}
	sleep.Install()
	sleep = examples.Sleep{"legacy"}
	sleep.InstallLegacy()

	util.Log.Info("Verify setup")
	for _, from := range []string{"foo", "bar", "legacy"} {
		for _, to := range []string{"foo", "bar"} {
			sleepPod, err := util.GetPodName(from, "app=sleep")
			util.Inspect(err, "Failed to get sleep pod name", "", t)
			cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
				to, from, to)
			msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true)
			util.Inspect(err, "Failed to get response", "", t)
			if !strings.Contains(msg, "200") {
				util.Log.Errorf("Verify setup -- Unexpected response code: %s", msg)
			} else {
				util.Log.Infof("Success. Get expected response: %s", msg)
			}
		}
	}

	t.Run("Security_authentication_namespace_enable_mtls", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Lock down to mutual TLS by namespace")
		util.KubeApplyContents("foo", NamespacePolicyStrict)
		time.Sleep(time.Duration(10) * time.Second)

		for _, from := range []string{"legacy"} {
			for _, to := range []string{"foo", "bar"} {
				sleepPod, err := util.GetPodName(from, "app=sleep")
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
					to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true)

				if from == "legacy" && to == "foo" {
					if err != nil {
						util.Log.Infof("Expected fail from sleep.legacy to httpbin.foo: %v", err)
					} else {
						t.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
						util.Log.Errorf("Expected fail from sleep.legacy to httpbin.foo; Got unexpected response: %s", msg)
					}
				} else {
					if !strings.Contains(msg, "200") {
						util.Log.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
						t.Errorf("Namespace mTLS expected: 200; Got unexpected response code: %s", msg)
					} else {
						util.Log.Infof("Success. Get expected response: %s", msg)
					}
				}
			}
		}
	})

	t.Run("Security_authentication_globally_enable_mtls", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Lock down to mutual TLS for the entire mesh")
		util.KubeApplyContents("istio-system", MeshPolicyStrict)
		time.Sleep(time.Duration(30) * time.Second)

		for _, from := range []string{"legacy"} {
			for _, to := range []string{"foo", "bar"} {
				sleepPod, err := util.GetPodName(from, "app=sleep")
				util.Inspect(err, "Failed to get sleep pod name", "", t)
				cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`, to, from, to)
				msg, err := util.PodExec(from, sleepPod, "sleep", cmd, true)
				if from == "legacy" && to == "foo" {
					if err != nil {
						util.Log.Infof("Expected sleep.legacy to httpbin.foo fails: %v", err)
					} else {
						t.Errorf("Expected sleep.legacy to httpbin.foo fails; Got unexpected response: %s", msg)
						util.Log.Errorf("Expected sleep.legacy to httpbin.foo fails; Got unexpected response: %s", msg)
					}
					continue
				}
				if from == "legacy" && to == "bar" {
					if err != nil {
						util.Log.Infof("Expected sleep.legacy to httpbin.bar fails: %v", err)
					} else {
						t.Errorf("Expected sleep.legacy to httpbin.bar fails; Got unexpected response: %s", msg)
						util.Log.Errorf("Expected sleep.legacy to httpbin.bar fails; Got unexpected response: %s", msg)
					}
					continue
				}
			}
		}
	})
}

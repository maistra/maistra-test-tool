// Copyright Red Hat, Inc.
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

package ossm

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
	"gopkg.in/yaml.v2"
)

func TestBasics(t *testing.T) {
	NewTest(t).Groups(ARM, Full, Smoke, InterOp).Run(func(t TestHelper) {
		t.Log("Test basics of SMCP: deploy, upgrade, bookinfo and uninstall")
		ns := "bookinfo"

		t.Cleanup(func() {
			app.Uninstall(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))
			oc.RecreateNamespace(t, meshNamespace)
		})

		toVersion := env.GetSMCPVersion()
		fromVersion := toVersion.GetPreviousVersion()

		oc.RecreateNamespace(t, meshNamespace)

		t.NewSubTest(fmt.Sprintf("install bookinfo with smcp %s", fromVersion)).Run(func(t TestHelper) {

			t.LogStepf("Create SMCP %s and verify it becomes ready", fromVersion)
			assertSMCPDeploysAndIsReady(t, fromVersion)

			t.LogStep("Install bookinfo pods with sidecar and sleep pod without sidecar")
			app.InstallAndWaitReady(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))

			t.LogStep("Check whether sidecar is injected in all bookinfo pods")
			assertSidecarInjectedInAllBookinfoPods(t, ns)

			t.LogStep("Check if bookinfo productpage is running through the Proxy")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns), "sleep",
					"curl -sI http://productpage:9080",
					assert.OutputContains(
						"HTTP/1.1 200 OK",
						"ProductPage returns 200 OK",
						"ProductPage didn't return 200 OK"),
					assert.OutputContains(
						"server: istio-envoy",
						"HTTP header 'server: istio-envoy' is present in the response",
						"HTTP header 'server: istio-envoy' is missing from the response"),
					assert.OutputContains(
						"x-envoy-decorator-operation",
						"HTTP header 'x-envoy-decorator-operation' is present in the response",
						"HTTP header 'x-envoy-decorator-operation' is missing from the response"))
			})

			t.LogStep("verify proxy startup time. Expected to be less than 5 seconds")
			t.Log("get proxy yaml and verify the time between status.containerStatuses.state.running.startedAt and status.conditions[type=Ready].lastTransitionTime")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				pod := pod.MatchingSelector("app=ratings", ns)(t, oc.DefaultOC)
				yaml := oc.GetYaml(t, ns, "pod", pod.Name)
				startedAt, lastTransitionTime := ExtractProxyTimes(t, yaml)
				diff := lastTransitionTime.Sub(startedAt)
				t.Logf("proxy startup time: %s", diff.String())
				if diff > 10*time.Second {
					t.Fatalf("proxy startup time is too long: %s", diff.String())
				}
			})
		})

		t.NewSubTest(fmt.Sprintf("upgrade %s to %s", fromVersion, toVersion)).Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP becomes ready after it's upgraded from %s to %s and bookinfo is still working after the upgrade", fromVersion, toVersion)

			t.LogStepf("Upgrade SMCP from %s to %s", fromVersion, toVersion)
			assertSMCPDeploysAndIsReady(t, toVersion)

			t.LogStep("Check if bookinfo productpage is running through the Proxy after the upgrade")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns), "sleep",
					"curl -sI http://productpage:9080",
					assert.OutputContains(
						"HTTP/1.1 200 OK",
						"ProductPage returns 200 OK",
						"ProductPage didn't return 200 OK"),
					assert.OutputContains(
						"server: istio-envoy",
						"HTTP header 'server: istio-envoy' is present in the response",
						"HTTP header 'server: istio-envoy' is missing from the response"),
					assert.OutputContains(
						"x-envoy-decorator-operation",
						"HTTP header 'x-envoy-decorator-operation' is present in the response",
						"HTTP header 'x-envoy-decorator-operation' is missing from the response"))
			})

			t.LogStep("Delete Bookinfo pods to validate proxy is still working after recreation and upgrade")
			oc.RestartAllPodsAndWaitReady(t, ns)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns), "sleep",
					"curl -sI http://productpage:9080",
					assert.OutputContains(
						"HTTP/1.1 200 OK",
						"ProductPage returns 200 OK",
						"ProductPage didn't return 200 OK"),
					assert.OutputContains(
						"server: istio-envoy",
						"HTTP header 'server: istio-envoy' is present in the response",
						"HTTP header 'server: istio-envoy' is missing from the response"),
					assert.OutputContains(
						"x-envoy-decorator-operation",
						"HTTP header 'x-envoy-decorator-operation' is present in the response",
						"HTTP header 'x-envoy-decorator-operation' is missing from the response"))
			})
		})

	})
}

func ExtractProxyTimes(t TestHelper, yamlString string) (time.Time, time.Time) {
	var data struct {
		Status struct {
			ContainerStatuses []struct {
				Name  string
				State struct {
					Running struct {
						StartedAt string
					}
				}
			}
			Conditions []struct {
				Type               string
				LastTransitionTime string
			}
		}
	}

	if err := yaml.Unmarshal([]byte(yamlString), &data); err != nil {
		t.Fatalf("Failed to unmarshal yaml: %s", err)
	}

	var startedAt, lastTransitionTime time.Time
	for _, status := range data.Status.ContainerStatuses {
		if status.Name == "istio-proxy" {
			startedAt, _ = time.Parse(time.RFC3339, status.State.Running.StartedAt)
			break
		}
	}

	for _, condition := range data.Status.Conditions {
		if condition.Type == "Ready" {
			lastTransitionTime, _ = time.Parse(time.RFC3339, condition.LastTransitionTime)
			break
		}
	}

	if startedAt.IsZero() || lastTransitionTime.IsZero() {
		t.Fatal("Failed to extract proxy times from yaml")
	}

	return startedAt, lastTransitionTime
}

func assertSidecarInjectedInAllBookinfoPods(t TestHelper, ns string) {
	shell.Execute(t,
		fmt.Sprintf(`oc -n %s get pods -l 'app in (productpage,details,reviews,ratings)' --no-headers`, ns),
		func(t TestHelper, input string) {
			scanner := bufio.NewScanner(strings.NewReader(input))
			for scanner.Scan() {
				line := scanner.Text()
				podName := strings.Fields(line)[0]
				if strings.Contains(line, "2/2") {
					t.LogSuccessf("Sidecar injected and running in pod %s", podName)
				} else {
					t.Errorf("Sidecar either not injected or not running in pod %s: %s", podName, line)
				}
			}
		})
}

func assertSMCPDeploysAndIsReady(t test.TestHelper, ver version.Version) {
	t.LogStep("Install SMCP")
	InstallSMCPVersion(t, meshNamespace, ver)
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
	oc.ApplyString(t, meshNamespace, GetSMMRTemplate())
	t.LogStep("Check SMCP is Ready")
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
}

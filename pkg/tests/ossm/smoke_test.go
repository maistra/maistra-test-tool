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

	"gopkg.in/yaml.v2"

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
)

func TestSmoke(t *testing.T) {
	NewTest(t).Groups(ARM, Full, Smoke, InterOp, Disconnected).Run(func(t TestHelper) {
		t.Log("Smoke Test for SMCP: deploy, upgrade, bookinfo and uninstall")
		ns := "bookinfo"

		t.Cleanup(func() {
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
				assertTrafficFlowsThroughProxy(t, ns)
			})

			t.LogStep("verify proxy startup time. Expected to be less than 10 seconds")
			t.Log("Jira related: https://issues.redhat.com/browse/OSSM-3586")
			t.Log("From proxy yaml, verify the time between status.containerStatuses.state.running.startedAt and status.conditions[type=Ready].lastTransitionTime")
			ratingPod := pod.MatchingSelector("app=ratings", ns)(t, oc.DefaultOC)
			ratingYaml := oc.GetYaml(t, ns, "pod", ratingPod.Name)
			t.Log("Validate from YAML, the proxy startup time is less than 10 seconds for ratings pod")
			validateStartUpProxyTime(t, ratingYaml)
		})

		t.NewSubTest(fmt.Sprintf("upgrade %s to %s", fromVersion, toVersion)).Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP becomes ready after it's upgraded from %s to %s and bookinfo is still working after the upgrade", fromVersion, toVersion)

			t.Cleanup(func() {
				app.Uninstall(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))
			})

			t.LogStepf("Upgrade SMCP from %s to %s", fromVersion, toVersion)
			assertSMCPDeploysAndIsReady(t, toVersion)

			t.LogStep("Check if bookinfo productpage is running through the Proxy after the upgrade")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertTrafficFlowsThroughProxy(t, ns)
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

		t.NewSubTest(fmt.Sprintf("delete smcp %s", toVersion)).Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP %s deletion delete all the resources", env.GetSMCPVersion())

			t.LogStep("Delete SMCP and verify if this deletes all resources")
			assertUninstallDeletesAllResources(t, env.GetSMCPVersion())
		})

		t.NewSubTest("verify continue working after smcp deletion").Run(func(t TestHelper) {
			t.Log("This test checks whether the dataplane still works after smcp deletion")

			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertTrafficFlowsThroughProxy(t, ns)
			})
		})
	})
}

func assertTrafficFlowsThroughProxy(t TestHelper, ns string) {
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
}

func validateStartUpProxyTime(t TestHelper, ratingYaml string) {
	proxyStartTime, proxyReadyTime := ExtractProxyTimes(t, ratingYaml)
	startupTime := proxyLastTransitionTime.Sub(proxyStartTime)
	t.Logf("Proxy startup time: %s", startupTime.String())
	if startupTime > 10*time.Second {
		t.Fatalf("Proxy startup time is too long: %s", startupTime.String())
	}
}

func ExtractProxyTimes(t TestHelper, yamlString string) (time.Time, time.Time) {
	var data struct {
		Status struct {
			ContainerStatuses []struct {
				Name  string
				State struct {
					Running struct {
						StartedAt string `yaml:"startedAt"`
					} `yaml:"running"`
				} `yaml:"state"`
			} `yaml:"containerStatuses"`
			Conditions []struct {
				Type               string
				LastTransitionTime string `yaml:"lastTransitionTime"`
			} `yaml:"conditions"`
		} `yaml:"status"`
	}

	err := yaml.Unmarshal([]byte(yamlString), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %s", err)
	}

	var startedAt, lastTransitionTime time.Time

	for _, status := range data.Status.ContainerStatuses {
		if status.Name == "istio-proxy" {
			startedAt, err = time.Parse(time.RFC3339, status.State.Running.StartedAt)
			if err != nil {
				t.Fatalf("Failed to parse startedAt time: %s", err)
			}
			break
		}
	}

	for _, condition := range data.Status.Conditions {
		if condition.Type == "Ready" {
			lastTransitionTime, err = time.Parse(time.RFC3339, condition.LastTransitionTime)
			if err != nil {
				t.Fatalf("Failed to parse lastTransitionTime time: %s", err)
			}
			break
		}
	}

	if startedAt.IsZero() || lastTransitionTime.IsZero() {
		t.Fatal("Failed to extract proxy times from YAML")
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

func assertUninstallDeletesAllResources(t test.TestHelper, ver version.Version) {
	t.LogStep("Delete SMCP in namespace " + meshNamespace)
	oc.DeleteFromString(t, meshNamespace, GetSMMRTemplate())
	DeleteSMCPVersion(t, meshNamespace, ver)
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.GetAllResources(t,
			meshNamespace,
			assert.OutputContains("No resources found in",
				"All resources deleted from namespace",
				"Still waiting for resources to be deleted from namespace"))
	})
}

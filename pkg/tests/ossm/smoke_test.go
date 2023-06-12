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
)

var (
	VERSIONS = []*version.Version{
		&version.SMCP_2_0,
		&version.SMCP_2_1,
		&version.SMCP_2_2,
		&version.SMCP_2_3,
		&version.SMCP_2_4,
	}
)

func TestSmoke(t *testing.T) {
	NewTest(t).Groups(ARM, Full, Smoke, InterOp, Disconnected).Run(func(t TestHelper) {
		t.Log("Smoke Test for SMCP: deploy, upgrade, bookinfo and uninstall")
		ns := "bookinfo"

		t.Cleanup(func() {
			app.Uninstall(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))
			oc.RecreateNamespace(t, meshNamespace)
		})

		toVersion := env.GetSMCPVersion()
		fromVersion := getPreviousVersion(toVersion)

		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Install bookinfo pods and sleep pod")
		app.InstallAndWaitReady(t, app.Bookinfo(ns), app.SleepNoSidecar(ns))

		t.NewSubTest(fmt.Sprintf("install bookinfo with smcp %s", fromVersion)).Run(func(t TestHelper) {

			t.LogStepf("Create SMCP %s and verify it becomes ready", fromVersion)
			assertSMCPDeploysAndIsReady(t, fromVersion)
			if env.GetSMCPVersion().GreaterThan(version.SMCP_2_2) {
				assertRoutesExist(t)
			}

			t.LogStep("Restart all pods to verify proxy is injected in all pods of Bookinfo")
			oc.RestartAllPods(t, ns)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				assertSidecarInjectedInAllBookinfoPods(t, ns)
			})

			t.LogStep("Check if bookinfo productpage is running through the Proxy")
			assertTrafficFlowsThroughProxy(t, ns)

			t.LogStep("verify proxy startup time. Expected to be less than 10 seconds")
			t.Log("Jira related: https://issues.redhat.com/browse/OSSM-3586")
			t.Log("From proxy json , verify the time between status.containerStatuses.state.running.startedAt and status.conditions[type=Ready].lastTransitionTime")
			t.Log("The proxy startup time should be less than 10 seconds for ratings pod")
			assertProxiesReadyInLessThan10Seconds(t, ns)
		})

		t.NewSubTest(fmt.Sprintf("upgrade %s to %s", fromVersion, toVersion)).Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP becomes ready after it's upgraded from %s to %s and bookinfo is still working after the upgrade", fromVersion, toVersion)

			t.LogStepf("Upgrade SMCP from %s to %s", fromVersion, toVersion)
			assertSMCPDeploysAndIsReady(t, toVersion)
			assertRoutesExist(t)

			t.LogStep("Check if bookinfo productpage is running through the Proxy after the upgrade")
			assertTrafficFlowsThroughProxy(t, ns)

			t.LogStep("Delete Bookinfo pods to validate proxy is still working after recreation and upgrade")
			oc.RestartAllPodsAndWaitReady(t, ns)
			assertTrafficFlowsThroughProxy(t, ns)
		})

		t.NewSubTest(fmt.Sprintf("delete smcp %s", toVersion)).Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP %s deletion deletes all the resources", env.GetSMCPVersion())

			t.LogStepf("Delete SMCP and SMMR in namespace %s", meshNamespace)
			oc.DeleteFromString(t, meshNamespace, GetSMMRTemplate())
			DeleteSMCPVersion(t, meshNamespace, env.GetSMCPVersion())
			t.LogStep("verify SMCP resources are deleted")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Get(t,
					meshNamespace,
					"smcp,pods,services", "",
					assert.OutputContains("No resources found in",
						"SMCP resources are deleted",
						"Still waiting for resources to be deleted from namespace"))
			})
		})

	})
}

func assertTrafficFlowsThroughProxy(t TestHelper, ns string) {
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
}

func assertProxiesReadyInLessThan10Seconds(t TestHelper, ns string) {
	t.Log("Extracting proxy startup time and last transition time for all the pods in the namespace")
	podsList := oc.GetJson(t, ns, "pods", "", `{.items[*].metadata.name}`)

	for _, podName := range strings.Split(podsList, " ") {
		// skip sleep pod because it doesn't have a proxy
		if strings.Contains(podName, "sleep") {
			continue
		}
		startedAt := oc.GetJson(t, ns, "pod", podName, `{.status.containerStatuses[?(@.name=="istio-proxy")].state.running.startedAt}`)
		readyAt := oc.GetJson(t, ns, "pod", podName, `{.status.conditions[?(@.type=="Ready")].lastTransitionTime}`)
		if startedAt != "" && readyAt != "" {
			podStartedAt, err := time.Parse(time.RFC3339, startedAt)
			if err != nil {
				t.Fatalf("Error parsing time for pod %d", podName)
			}
			podReadyAt, err := time.Parse(time.RFC3339, readyAt)
			if err != nil {
				t.Fatalf("Error parsing time for pod %d", podName)
			}
			startupTime := podReadyAt.Sub(podStartedAt)
			if startupTime > 10*time.Second {
				t.Fatalf("Proxy startup time is too long: %s", startupTime.String())
			}
		} else {
			t.Fatalf("Error getting proxy startup time for pod %s", podName)
		}
	}
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

func assertRoutesExist(t test.TestHelper) {
	t.LogStep("Verify if all the routes are created")
	t.Log("Related issue: https://issues.redhat.com/browse/OSSM-4069")
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.Get(t,
			meshNamespace,
			"routes", "",
			assert.OutputContains("grafana",
				"Route grafana is created",
				"Still waiting for route grafana to be created in namespace"),
			assert.OutputContains("istio-ingressgateway",
				"Route istio-ingressgateway is created",
				"Still waiting for route istio-ingressgateway to be created in namespace"),
			assert.OutputContains("jaeger",
				"Route jaeger is created",
				"Still waiting for route jaeger to be created in namespace"),
			assert.OutputContains("kiali",
				"Route kiali is created",
				"Still waiting for route kiali to be created in namespace"),
			assert.OutputContains("prometheus",
				"Route prometheus is created",
				"Still waiting for route prometheus to be created in namespace"))
	})
}

func getPreviousVersion(ver version.Version) version.Version {
	var prevVersion *version.Version
	for _, v := range VERSIONS {
		if *v == ver {
			if prevVersion == nil {
				panic(fmt.Sprintf("version %s is the first supported version", ver))
			}
			return *prevVersion
		}
		prevVersion = v
	}
	panic(fmt.Sprintf("version %s not found in VERSIONS", ver))
}

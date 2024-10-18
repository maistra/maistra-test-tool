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

package ossm

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/addons-route-patch.yaml
	addonsCustomRoutesPatch string
)

func TestSMCPAddons(t *testing.T) {
	NewTest(t).Id("T34").Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		DeployControlPlane(t)
		// Created a subtest because we need to add more test related to Addons in the future.
		t.NewSubTest("3scale_addon").Run(func(t TestHelper) {
			t.LogStep("Enable 3scale in a SMCP expecting to get validation error.")
			t.Cleanup(func() {
				shell.Execute(t, fmt.Sprintf(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"addons":{"3scale":{"enabled":false}}}}' || true`, meshNamespace, smcpName))
			})

			shell.Execute(t,
				fmt.Sprintf(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"addons":{"3scale":{"enabled":true}}}}' || true`, meshNamespace, smcpName),
				assert.OutputContains("support for 3scale has been removed",
					"Got expected validation error: support for 3scale has been removed",
					"The validation error was not shown as expected"))
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.NewSubTest("addons_custom_routes").Run(func(t TestHelper) {
			// skip for kiali, OSSM-3026 is not resolved
			t.LogStep("Set custom routes for grafana and prometheus addons")
			t.Log("See https://issues.redhat.com/browse/OSSM-534")
			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
			})
			t.LogStep("Set custom routes for addons")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", addonsCustomRoutesPatch)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.WaitKialiReady(t, meshNamespace, "kiali")
			// TODO check custom routes
			for _, addon := range []string{"grafana", "prometheus"} {
				route := oc.DefaultOC.GetRouteURL(t, meshNamespace, addon)
				if route != fmt.Sprintf("test.%s.com", addon) {
					t.Errorf("Addon %s doesn't have expected custom route, instead, it has %s", addon, route)
				}
			}
		})

		t.NewSubTest("disable_addons").Run(func(t TestHelper) {
			t.Log("This test checks if the pods are removed when addons are disabled")
			t.Log("See https://issues.redhat.com/browse/OSSM-1490")
			meshNamespace := env.GetDefaultMeshNamespace()
			smcpName := env.GetDefaultSMCPName()

			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
			})

			t.LogStep("Install SMCP and wait for it to be Ready")
			DeployControlPlane(t)
			oc.WaitKialiReady(t, meshNamespace, "kiali")

			t.LogStep("Wait till Grafana/Prometheus/Kiali pods are ready")
			oc.WaitPodRunning(t, pod.MatchingSelector("app=kiali", meshNamespace))
			oc.WaitPodRunning(t, pod.MatchingSelector("app=prometheus", meshNamespace))
			oc.WaitPodRunning(t, pod.MatchingSelector("app=grafana", meshNamespace))

			t.LogStep("Disable Grafana/Kiali addons")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			t.LogStep("Check that Grafana/Kiali pods were deleted but not Prometheus")
			checkThatPodWasDeleted(t, meshNamespace, "grafana")
			checkThatPodExist(t, meshNamespace, "prometheus")
			checkThatPodWasDeleted(t, meshNamespace, "kiali")

			t.LogStep("Disable also Prometheus")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  addons:
    prometheus:
      enabled: false
`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			t.LogStep("Check that all addons pods were deleted")
			checkThatPodWasDeleted(t, meshNamespace, "kiali")
			checkThatPodWasDeleted(t, meshNamespace, "grafana")
			checkThatPodWasDeleted(t, meshNamespace, "prometheus")
		})
	})
}

func checkThatPodWasDeleted(t TestHelper, ns string, nameSelector string) {
	retry.UntilSuccess(t, func(t TestHelper) {
		if oc.ResourceByLabelExists(t, ns, "pod", "app="+nameSelector) {
			t.Errorf("Pod with label app=%s still exists", nameSelector)
		}
	})

}

func checkThatPodExist(t TestHelper, ns string, nameSelector string) {
	if !oc.ResourceByLabelExists(t, ns, "pod", "app="+nameSelector) {
		t.Errorf("Pod with label app=%s doesn't exist", nameSelector)
	}
}

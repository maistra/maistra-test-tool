// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAddonsDisable(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.Disconnected, test.ARM).Run(func(t test.TestHelper) {
		t.Log("This test checks if the pods are removed when addons are disabled")
		t.Log("See https://issues.redhat.com/browse/OSSM-1490")
		meshNamespace := env.GetDefaultMeshNamespace()
		smcpName := env.GetDefaultSMCPName()

		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStep("Install SMCP and wait for it to be Ready")
		ossm.DeployControlPlane(t)
		oc.WaitCondition(t, meshNamespace, "Kiali", "kiali", "Successful")

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
}

func checkThatPodWasDeleted(t test.TestHelper, ns string, nameSelector string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		if oc.ResourceByLabelExists(t, ns, "pod", "app="+nameSelector) {
			t.Errorf("Pod with label app=%s still exists", nameSelector)
		}
	})

}

func checkThatPodExist(t test.TestHelper, ns string, nameSelector string) {
	if !oc.ResourceByLabelExists(t, ns, "pod", "app="+nameSelector) {
		t.Errorf("Pod with label app=%s doesn't exist", nameSelector)
	}
}

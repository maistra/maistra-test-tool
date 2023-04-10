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
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/multiple-namespaces.yaml
	multiple_namespaces string

	//go:embed yaml/smmr-test-multiple-members.yaml
	smmr_multiple_members string
)

// TestIstiodPodFailsAfterRestarts tests that Istio pod get stuck with probes failure after restart. Jira ticket bug: https://issues.redhat.com/browse/OSSM-2434
func TestIstiodPodFailsAfterRestarts(t *testing.T) {
	NewTest(t).Id("T35").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		data := map[string]interface{}{
			"Count":     50,
			"Namespace": meshNamespace,
		}

		t.Cleanup(func() {
			deleteMultipleNamespaces(t, data)
			oc.RecreateNamespace(t, meshNamespace)
			oc.ApplyString(t, meshNamespace, smmr)
		})

		t.LogStep("Create Namespaces and SMMR")
		createMultipleNamespaces(t, data)
		updateSMMRMultipleNamespaces(t, meshNamespace, smmr_multiple_members, data)

		t.LogStep("Delete Istio pod 10 times and check that it is running and ready after the deletions")
		for i := 0; i < 10; i++ {
			oc.DeletePod(t, pod.MatchingSelector("app=istiod", meshNamespace))
			oc.WaitPodRunning(t, pod.MatchingSelector("app=istiod", meshNamespace))
			oc.WaitPodReady(t, pod.MatchingSelector("app=istiod", meshNamespace))
		}
	})
}

func updateSMMRMultipleNamespaces(t test.TestHelper, ns string, yaml string, data interface{}) {
	t.T().Helper()
	t.Logf("Creating smmr")
	oc.ApplyTemplate(t, ns, yaml, data)
}

func createMultipleNamespaces(t test.TestHelper, data interface{}) {
	t.T().Helper()
	t.Logf("Creating multiple namespaces: %s", data.(map[string]interface{})["Count"])
	oc.ApplyTemplate(t, "default", multiple_namespaces, data)
}

func deleteMultipleNamespaces(t test.TestHelper, data interface{}) {
	t.T().Helper()
	t.Logf("Deleting multiple namespaces")
	oc.DeleteFromTemplate(t, "default", multiple_namespaces, data)
}

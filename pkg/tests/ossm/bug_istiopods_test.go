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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestIstioPodProbesFails tests that Istio pod get stuck with probes failure after restart. Jira ticket bug: https://issues.redhat.com/browse/OSSM-2434
func TestIstioPodProbesFails(t *testing.T) {
	NewTest(t).Id("T35").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "bookinfo"
		data := map[string]interface{}{
			"Count":     50,
			"Namespace": meshNamespace,
		}

		t.Cleanup(func() {
			oc.DeleteNamespaces(t, multiple_namespaces, data)
			oc.DeleteNamespace(t, ns)
			oc.RecreateNamespace(t, meshNamespace)

		})

		t.LogStep("Install Bookinfo application")
		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		t.LogStep("Create Namespaces and SMMR")
		oc.CreateNamespaces(t, multiple_namespaces, data)
		oc.UpdateSMMRMultipleNamespaces(t, meshNamespace, multiple_smmr, data)

		t.LogStep("Delete Istio pod and check that it is running again")
		assertIstiodPodReadyAfterDeletion(t, meshNamespace, 10)
	})
}

func assertIstiodPodReadyAfterDeletion(t test.TestHelper, ns string, deletionTimes int) {
	for i := 0; i < deletionTimes; i++ {
		oc.DeletePod(t, pod.MatchingSelector("app=istiod", ns))
		oc.WaitPodRunning(t, pod.MatchingSelector("app=istiod", ns))
		oc.WaitPodReady(t, pod.MatchingSelector("app=istiod", ns))
	}
}

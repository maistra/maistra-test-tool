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
	"fmt"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestIstiodPodFailsAfterRestarts tests that Istio pod get stuck with probes failure after restart. Jira ticket bug: https://issues.redhat.com/browse/OSSM-2434
func TestIstiodPodFailsAfterRestarts(t *testing.T) {
	NewTest(t).Id("T35").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)

		namespaces := util.GenerateStrings("test-", 50)

		t.Cleanup(func() {
			oc.DeleteNamespace(t, namespaces...)
			oc.RecreateNamespace(t, meshNamespace)
			oc.ApplyString(t, meshNamespace, smmr)
		})

		t.LogStep("Create Namespaces and SMMR")
		oc.CreateNamespace(t, namespaces...)
		oc.ApplyString(t, meshNamespace, createSMMRManifest(namespaces...))

		t.LogStep("Delete Istio pod 10 times and check that it is running and ready after the deletions")
		for i := 0; i < 10; i++ {
			istiodPod := pod.MatchingSelector("app=istiod", meshNamespace)
			oc.DeletePod(t, istiodPod)
			oc.WaitPodRunning(t, istiodPod)
			oc.WaitPodReady(t, istiodPod)
		}
	})
}

func createSMMRManifest(namespaces ...string) string {
	return fmt.Sprintf(`
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
    - bookinfo
    - foo
    - bar
    - legacy%s`, strings.Join(namespaces, "\n    - "))
}

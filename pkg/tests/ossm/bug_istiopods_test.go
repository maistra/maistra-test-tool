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
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestIstiodPodFailsAfterRestarts(t *testing.T) {
	NewTest(t).Id("T35").Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		t.Log("Verify that Istio pod not get stuck with probes failure after restart")
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-2434")
		namespaces := util.GenerateStrings("test-", 50)

		t.Cleanup(func() {
			oc.DeleteNamespace(t, namespaces...)
			oc.ApplyString(t, meshNamespace, smmr)
		})

		DeployControlPlane(t) // integrate below SMMR stuff here

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

func TestIstiodPodFailsWithValidationMessages(t *testing.T) {
	NewTest(t).Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		t.Log("Verify that Istio pod is not failing when validationMessages was enabled")

		oc.RecreateNamespace(t, meshNamespace)
		oc.ReplaceOrApplyString(t, meshNamespace, template.Run(t, validationMessagesSMCP, DefaultSMCP()))
		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})

		istiodPod := pod.MatchingSelector("app=istiod", meshNamespace)
		oc.WaitPodRunning(t, istiodPod)
		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(10), func(t TestHelper) {
			oc.LogsFromPods(t, meshNamespace, "app=istiod", assert.OutputContains(
				"successfully acquired lease "+meshNamespace+"/istio-analyze-leader",
				"Successfully acquired lease for analyzer in istiod pod",
				"Expected to acquire lease for analyzer in istiod pod, but was not",
			),
			)
		})
		time.Sleep(time.Second * 5) //wait 5 seconds to make sure that the errors appear after the lease is acquired

		t.Log("Verify that Istiod pod doesn't contain SIGSEGV when validationMessages was enabled")
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-6177")
		oc.Logs(t,
			istiodPod,
			"discovery",
			assert.OutputDoesNotContain(
				"SIGSEGV: segmentation violation",
				"Found no SIGSEGV",
				"Expected to find no SIGSEGV, but got some SIGSEGV",
			),
		)

		t.Log("Verify that istiod doesn't contain any cannot list resource error when validationMessages was enabled")
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-6289")
		oc.Logs(t,
			istiodPod,
			"discovery",
			assert.OutputDoesNotContain(
				"watch error in cluster : failed to list",
				"Found no `cannot list resource` error",
				"Expected to find no `cannot list resource` error, but got some error",
			),
		)
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

const (
	validationMessagesSMCP = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: {{ .Version }}
  general:
    validationMessages: true 
  tracing:
    type: None
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  {{ if .Rosa }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}`
)

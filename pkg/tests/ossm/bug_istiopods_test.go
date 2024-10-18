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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
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

func TestControllerFailsToUpdatePod(t *testing.T) {
	NewTest(t).Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		t.Log("Verify that the controller does not fails to update the pod when the member controller couldn't add the member-of label")
		t.Log("References: \n- https://issues.redhat.com/browse/OSSM-2169\n- https://issues.redhat.com/browse/OSSM-2420")

		// Add timestamp to the namespaces to avoid conflicts during tests execution
		ti := time.Now()
		nameTemplate := fmt.Sprintf("test-smcp-%d%d-%02d%02d-",
			env.GetSMCPVersion().Major,
			env.GetSMCPVersion().Minor,
			ti.Hour(), ti.Minute())

		namespaces := util.GenerateStrings(nameTemplate, 100)
		re := regexp.MustCompile(fmt.Sprintf(`error adding member-of label to namespace (%s\d+)`, nameTemplate))

		t.Cleanup(func() {
			oc.DeleteNamespace(t, namespaces...)
			oc.ApplyString(t, meshNamespace, smmr)
		})

		DeployControlPlane(t)

		t.LogStep("Add namespaces to the SMMR")
		oc.ApplyString(t, meshNamespace, createSMMRManifest(namespaces...))

		istioOperatorPodName := oc.GetAllResoucesNamesByLabel(t, "openshift-operators", "pod", "name=istio-operator")[0]

		// Initially assumed 1 iteration, however, as issue can be flaky it could be increased up to 5 iterations
		count := 2
		for i := 1; i < count; i++ {
			t.LogStepf("Create/Recreate 100 Namespaces, attempt #%d", i)
			oc.RecreateNamespace(t, namespaces...)

			t.LogStepf("Check istio-operator logs for 'Error updating pod's labels', attempt #%d", i)
			output := shell.Execute(t,
				fmt.Sprintf("oc logs %s -n openshift-operators", istioOperatorPodName),
				assert.OutputDoesNotContain(
					"Error updating pod's labels",
					"Found no updating pod's labels error",
					"Expected to find no error updating pod's labels, but got",
				))

			t.LogStepf("Check istio-operator logs for 'error adding member-of label' errors, attempt #%d", i)
			matches := re.FindStringSubmatch(output)
			if len(matches) > 1 {
				namespaceName := matches[1]
				successMessage := fmt.Sprintf(`Added member-of label to namespace","ServiceMeshMember":"%s/default","namespace":"%s`, namespaceName, namespaceName)
				if strings.Contains(output, successMessage) {
					t.LogSuccessf("Found error and success message for namespace %s: %s", namespaceName, successMessage)
					break
				} else {
					t.Log(output)
					t.Fatalf("Was not found success message after error for namespace %s: %s", namespaceName, successMessage)
				}
			} else {
				if count < 6 {
					count++
					t.Logf("Was not found any 'error adding member-of label' error, repeat (max 5), attempt #%d", i)
				} else {
					t.Logf("Was not found any 'error adding member-of label' error, stop test, attempt #%d", i)
				}
			}
		}
	})
}

func TestIstiodPodFailsWithValidationMessages(t *testing.T) {
	NewTest(t).Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		t.Log("Verify that Istio pod is not failing when validationMessages was enabled")

		oc.RecreateNamespace(t, meshNamespace)
		oc.ApplyString(t, meshNamespace, template.Run(t, validationMessagesSMCP, DefaultSMCP()))
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
    - legacy
    - %s`, strings.Join(namespaces, "\n    - "))
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

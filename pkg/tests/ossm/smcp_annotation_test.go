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
	"encoding/json"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/deployment-testenv-x86.yaml
	testenvDeploymentX86 string

	//go:embed yaml/deployment-testenv-z.yaml
	testenvDeploymentZ string

	//go:embed yaml/deployment-testenv-p.yaml
	testenvDeploymentP string

	InjectedAnnotationsSMCPPatch = `{"spec":{"proxy":{"injection":{"autoInject":true,"injectedAnnotations":{"test1.annotation-from-smcp":"test1","test2.annotation-from-smcp":"[\"test2\"]","test3.annotation-from-smcp":"{test3}"}}}}}`
)

func TestSMCPAnnotations(t *testing.T) {
	NewTest(t).Id("T29").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "foo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.NewSubTest("proxyEnvoy").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.RecreateNamespace(t, ns)
			})
			t.LogStep("Deploy TestSSL pod with annotations sidecar.maistra.io/proxyEnv")
			DeployTestSsl(t, ns)

			t.LogStep("Get annotations and verify that the pod has the expected ones")
			annotations := map[string]string {
				"sidecar.maistra.io/proxyEnv": `{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }`
			}
			pod := pod.MatchingSelector("app=env", ns)
			podAnnotations := GetPodAnnotations(t, pod)
			for k, v := range annotations {
				t.Logf("Checking annotation %s=%s", k, v)
				if podAnnotations[k] != v {
					t.Fatalf("Expected annotation %s=%s, but got %s", k, v, podAnnotations[k])
				}
			}
		})

		// Test that the SMCP automatic injection with quotes works
		t.NewSubTest("quote_injection").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.RecreateNamespace(t, ns)
			})
			t.LogStep("Enable annotation auto injection in SMCP")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", InjectedAnnotationsSMCPPatch)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			t.LogStep("Deploy TestSSL pod with annotations sidecar.maistra.io/proxyEnv")
			DeployTestSsl(t, ns)

			t.LogStep("Get annotations and verify that the pod has the expected ones")
			annotations := make(map[string]string)
			annotations["test1.annotation-from-smcp"] = "test1"
			annotations["test2.annotation-from-smcp"] = `["test2"]`
			annotations["test3.annotation-from-smcp"] = "{test3}"
			pod := pod.MatchingSelector("app=env", ns)
			podAnnotations := GetPodAnnotations(t, pod)
			for k, v := range annotations {
				if podAnnotations[k] != v {
					t.Fatalf("Expected annotation %s=%s, but got %s", k, v, podAnnotations[k])
				}
			}
		})

	})
}

func DeployTestSsl(t TestHelper, ns string) {
	yaml := getTestSsslYaml()
	oc.ApplyString(t, ns, yaml)
	operatorPod := pod.MatchingSelector("app=env", ns)
	oc.WaitPodRunning(t, operatorPod)
}

func getTestSsslYaml() string {
	yaml := ""
	switch env.Getenv("SAMPLEARCH", "x86") {
	case "p":
		yaml = testenvDeploymentP
	case "z":
		yaml = testenvDeploymentZ
	default:
		yaml = testenvDeploymentX86
	}
	return yaml
}

func GetPodAnnotations(t TestHelper, podLocator oc.PodLocatorFunc) map[string]string {
	annotations := map[string]string{}
	pod := podLocator(t)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		output := shell.Execute(t, fmt.Sprintf("kubectl get pod %s -n %s -o jsonpath='{.metadata.annotations}'", pod.Name, pod.Namespace))
		json.Unmarshal([]byte(output), &annotations)
	})
	return annotations
}

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

func TestSMCPAnnotations(t *testing.T) {
	NewTest(t).Id("T29").Groups(Full).Run(func(t TestHelper) {
		t.Log("Test annotations: verify deployment with sidecar.maistra.io/proxyEnv annotations and Enable automatic injection in SMCP to propagate the annotations to the sidecar")
		hack.DisableLogrusForThisTest(t)

		t.NewSubTest("proxyEnvoy").Run(func(t TestHelper) {
			t.Parallel()
			ns := "foo"
			t.Cleanup(func() {
				oc.RecreateNamespace(t, ns)
			})
			t.LogStep("Deploy TestSSL pod with annotations sidecar.maistra.io/proxyEnv")
			oc.ApplyString(t, ns, getTestSSLManifestWithAnnotation())
			oc.WaitDeploymentRolloutComplete(t, ns, "testenv")

			t.LogStep("Get annotations and verify that the pod has the expected: sidecar.maistra.io/proxyEnv : { \"maistra_test_env\": \"env_value\", \"maistra_test_env_2\": \"env_value_2\" }")
			annotations := GetPodAnnotations(t, pod.MatchingSelector("app=env", ns))
			assertAnnotationIsPresent(t, annotations, "sidecar.maistra.io/proxyEnv", `{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }`)
		})

		// Test that the SMCP automatic injection with quotes works
		t.NewSubTest("quote_injection").Run(func(t TestHelper) {
			t.Parallel()
			ns := "bar"
			t.Cleanup(func() {
				oc.RecreateNamespace(t, ns)
			})
			t.LogStep("Enable annotation auto injection in SMCP")
			oc.Patch(t,
				meshNamespace,
				"smcp", smcpName,
				"merge",
				`{"spec":{"proxy":{"injection":{"autoInject":true,"injectedAnnotations":{"test1.annotation-from-smcp":"test1","test2.annotation-from-smcp":"[\"test2\"]","test3.annotation-from-smcp":"{test3}"}}}}}`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			t.LogStep("Deploy TestSSL pod with annotations sidecar.maistra.io/proxyEnv")
			oc.ApplyString(t, ns, getTestSSLManifestWithAnnotation())
			oc.WaitDeploymentRolloutComplete(t, ns, "testenv")

			t.LogStep("Get annotations and verify that the pod has the expected: test1.annotation-from-smcp : test1, test2.annotation-from-smcp : [\"test2\"], test3.annotation-from-smcp : {test3}")
			annotations := GetPodAnnotations(t, pod.MatchingSelector("app=env", ns))
			assertAnnotationIsPresent(t, annotations, "test1.annotation-from-smcp", "test1")
			assertAnnotationIsPresent(t, annotations, "test2.annotation-from-smcp", `["test2"]`)
			assertAnnotationIsPresent(t, annotations, "test3.annotation-from-smcp", "{test3}")
		})
	})
}

func GetPodAnnotations(t TestHelper, podLocator oc.PodLocatorFunc) map[string]string {
	annotations := map[string]string{}
	po := podLocator(t, oc.DefaultOC)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		output := shell.Executef(t, "kubectl get pod %s -n %s -o jsonpath='{.metadata.annotations}'", po.Name, po.Namespace)
		err := json.Unmarshal([]byte(output), &annotations)
		if err != nil {
			t.Fatalf("Error parsing pod annotations json: %v", err)
		}
	})
	return annotations
}

func assertAnnotationIsPresent(t TestHelper, annotations map[string]string, key string, expectedValue string) {
	if annotations[key] != expectedValue {
		t.Fatalf("Expected annotation %s=%s, but got %s", key, expectedValue, annotations[key])
	}
}

func getTestSSLManifestWithAnnotation() string {
	return fmt.Sprintf(testSSLDeploymentWithAnnotation, env.GetTestSSLImage())
}

const testSSLDeploymentWithAnnotation = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: testenv
spec:
  replicas: 1
  selector:
    matchLabels:
      app: env
  template:
    metadata:
      annotations:
        sidecar.maistra.io/proxyEnv: '{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }'
      labels:
        app: env
    spec:
      terminationGracePeriodSeconds: 0
      containers:
      - name: testenv
        image: %s
`

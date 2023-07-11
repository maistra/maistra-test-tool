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

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSMCPAnnotations(t *testing.T) {
	test.NewTest(t).Id("T29").Groups(test.Full, test.Disconnected).Run(func(t test.TestHelper) {
		t.Log("Test annotations: verify deployment with sidecar.maistra.io/proxyEnv annotations and Enable automatic injection in SMCP to propagate the annotations to the sidecar")

		DeployControlPlane(t) // TODO: move this to individual subtests and integrate patch if one exists

		t.NewSubTest("proxyEnvoy").Run(func(t test.TestHelper) {
			t.Parallel()
			ns := "foo"
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, testSSLDeploymentWithAnnotation, nil)
			})
			t.LogStep("Deploy TestSSL pod with annotations sidecar.maistra.io/proxyEnv")
			oc.ApplyTemplate(t, ns, testSSLDeploymentWithAnnotation, nil)
			oc.WaitDeploymentRolloutComplete(t, ns, "testenv")

			t.LogStep("Get annotations and verify that the pod has the expected: sidecar.maistra.io/proxyEnv : { \"maistra_test_env\": \"env_value\", \"maistra_test_env_2\": \"env_value_2\" }")
			annotations := VerifyAndGetPodAnnotation(t, pod.MatchingSelector("app=env", ns))
			assertAnnotationIsPresent(t, annotations, "sidecar.maistra.io/proxyEnv", `{ "maistra_test_env": "env_value", "maistra_test_env_2": "env_value_2" }`)
		})

		// Test that the SMCP automatic injection with quotes works
		t.NewSubTest("quote_injection").Run(func(t test.TestHelper) {
			t.Parallel()
			ns := "bar"
			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json", `[{"op": "remove", "path": "/spec/proxy"}]`)
				oc.DeleteFromTemplate(t, ns, testSSLDeploymentWithAnnotation, nil)
			})
			t.LogStep("Enable annotation auto injection in SMCP")
			oc.Patch(t,
				meshNamespace,
				"smcp", smcpName,
				"merge",
				`{"spec":{"proxy":{"injection":{"autoInject":true,"injectedAnnotations":{"test1.annotation-from-smcp":"test1","test2.annotation-from-smcp":"[\"test2\"]","test3.annotation-from-smcp":"{test3}"}}}}}`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			t.LogStep("Deploy TestSSL pod with annotations sidecar.maistra.io/proxyEnv")
			oc.ApplyTemplate(t, ns, testSSLDeploymentWithAnnotation, nil)
			oc.WaitDeploymentRolloutComplete(t, ns, "testenv")

			t.LogStep("Get annotations and verify that the pod has the expected: test1.annotation-from-smcp : test1, test2.annotation-from-smcp : [\"test2\"], test3.annotation-from-smcp : {test3}")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				annotations := VerifyAndGetPodAnnotation(t, pod.MatchingSelector("app=env", ns))
				assertAnnotationIsPresent(t, annotations, "test1.annotation-from-smcp", "test1")
				assertAnnotationIsPresent(t, annotations, "test2.annotation-from-smcp", `["test2"]`)
				assertAnnotationIsPresent(t, annotations, "test3.annotation-from-smcp", "{test3}")
			})
		})
	})
}

func VerifyAndGetPodAnnotation(t test.TestHelper, podLocator oc.PodLocatorFunc) map[string]string {
	var data struct {
		Metadata struct {
			Annotations map[string]string `yaml:"annotations"`
		} `yaml:"metadata"`
	}

	po := podLocator(t, oc.DefaultOC)
	yamlString := oc.GetYaml(t, po.Namespace, "pod", po.Name)
	err := yaml.Unmarshal([]byte(yamlString), &data)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %s", err)
	}

	annotations := data.Metadata.Annotations
	if len(annotations) == 0 {
		oc.DeletePod(t, podLocator)
		oc.WaitPodReady(t, podLocator)
		t.Fatalf("Failed to get annotations from pod %s", po.Name)
	}

	return annotations
}

func assertAnnotationIsPresent(t test.TestHelper, annotations map[string]string, key string, expectedValue string) {
	locator := pod.MatchingSelector("app=env", "foo")
	if annotations[key] != expectedValue {
		oc.DeletePod(t, locator)
		oc.WaitPodReady(t, locator)
		t.Fatalf("Expected annotation %s=%s, but got %s", key, expectedValue, annotations[key])
	}
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
        image: {{ image "testssl" }}
`

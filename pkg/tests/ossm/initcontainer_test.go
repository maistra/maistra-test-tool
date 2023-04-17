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

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestInitContainerNotRemovedDuringInjection(t *testing.T) {
	const ns = "bookinfo"
	const goldString = "[init worked]"
	const podSelector = "app=sleep-init"

	NewTest(t).Id("T33").Groups(Full).Run(func(t TestHelper) {
		t.Log("Checking init container not removed during sidecar injection.")

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		oc.RecreateNamespace(t, ns)

		t.LogStep("Deploying test pod.")
		oc.ApplyString(t, ns, testInitContainerYAML)
		oc.WaitDeploymentRolloutComplete(t, ns, "sleep-init")

		t.LogStep("Checking pod logs for init message.")
		oc.Logs(t,
			pod.MatchingSelector(podSelector, ns),
			"init",
			assert.OutputContains(goldString,
				"Init container executed successfully.",
				"Init container did not execute."))
	})
}

const (
	testInitContainerYAML = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sleep-init
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sleep-init
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "true"
      labels:
        app: sleep-init
    spec:
      terminationGracePeriodSeconds: 0

      initContainers:
      - name: init
        image: curlimages/curl
        command: ["/bin/echo", "[init worked]"]
        imagePullPolicy: IfNotPresent

      containers:
      - name: sleep
        image: curlimages/curl
        command: ["/bin/sleep", "3650d"]
        imagePullPolicy: IfNotPresent`
)

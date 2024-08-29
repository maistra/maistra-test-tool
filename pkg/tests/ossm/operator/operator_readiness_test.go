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
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestOperatorPodHonorsReadinessProbe(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.Disconnected, test.ARM).MinVersion(version.SMCP_2_4).Run(func(t test.TestHelper) {
		t.Log("This test checks if the operator correctly reports its readiness status")

		meshNamespace := env.GetDefaultMeshNamespace()
		operatorPod := pod.MatchingSelector("name=istio-operator", env.GetOperatorNamespace())

		t.LogStep("Install SMCP and wait for it to be Ready")
		ossm.InstallSMCP(t, meshNamespace)
		oc.WaitSMCPReady(t, meshNamespace, env.GetDefaultSMCPName())

		t.LogStep("Delete istio-operator pod")
		oc.DeletePodNoWait(t, operatorPod)

		t.LogStep("Wait for pod to start running")
		oc.WaitPodRunning(t, operatorPod)

		t.LogStep("Confirm pod is not yet ready")
		shell.Execute(t,
			fmt.Sprintf("oc -n %s get po -l name=istio-operator", env.GetOperatorNamespace()),
			assert.OutputContains("0/1",
				"pod running, but not yet ready",
				"expected pod to not be ready immediately after starting up, but became ready immediately"))

		t.LogStep("Wait for pod to be ready")
		oc.WaitPodReady(t, operatorPod)

		t.LogStep("Check if readiness probe responds to request")
		oc.Exec(t, operatorPod, "istio-operator",
			"curl -sSI localhost:11200/readyz/",
			assert.OutputContains(
				"200",
				"readiness probe responds with 200 OK",
				"expected readiness probe to respond with 200 OK, but received a different response"))
	})
}

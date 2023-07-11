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

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var workername string

func TestDeployOnInfraNodes(t *testing.T) {
	test.NewTest(t).Id("T40").Groups(test.Full, test.Disconnected).Run(func(t test.TestHelper) {
		t.Log("This test verifies that the OSSM operator and Istio components can be configured to run on infrastructure nodes")

		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.Skip("Deploy On Infra node is available in SMCP versions v2.3+")
		}
		if env.IsRosa() {
			t.Skip("Skipping test on ROSA due to lack of permissions")
		}
		output := shell.Executef(t, `oc get subscription -n %s servicemeshoperator || true`, env.GetOperatorNamespace())
		if strings.Contains(output, "NotFound") {
			t.Skip("Skipping test because servicemeshoperator Subscription wasn't found")
		}

		t.Cleanup(func() {
			oc.Patch(t, env.GetOperatorNamespace(), "subscription", "servicemeshoperator", "json", `[{"op": "remove", "path": "/spec/config"}]`)
			oc.TaintNode(t, "-l node-role.kubernetes.io/infra",
				"node-role.kubernetes.io/infra=reserved:NoSchedule-",
				"node-role.kubernetes.io/infra=reserved:NoExecute-")
			oc.Label(t, "", "node", workername, "node-role.kubernetes.io/infra-")
			oc.Label(t, "", "node", workername, "node-role.kubernetes.io-")
			//TODO: to improve this after merge this PR: https://github.com/maistra/maistra-test-tool/pull/597 we need to reuse waitOperatorSucceded and make that func available to all the code
			// Use waitOperatorSucceded will avoid flaky in this test case
			locator := pod.MatchingSelector("name=istio-operator", "openshift-operators")
			oc.WaitPodReady(t, locator)
		})

		t.LogStep("Setup: Get a worker node from the cluster that does not have the istio operator installed and label it as infra")
		workername = pickWorkerNode(t)
		t.Logf("Worker node selected: %s", workername)
		oc.Label(t, "", "node", workername, "node-role.kubernetes.io/infra=")
		oc.Label(t, "", "node", workername, "node-role.kubernetes.io=infra")
		oc.TaintNode(t, "-l node-role.kubernetes.io/infra",
			"node-role.kubernetes.io/infra=reserved:NoSchedule",
			"node-role.kubernetes.io/infra=reserved:NoExecute")

		t.NewSubTest("operator").Run(func(t test.TestHelper) {
			t.Log("Verify OSSM Operator is deployed on infra node when configured")
			t.Log("Reference: https://issues.redhat.com/browse/OSSM-2342")

			t.LogStep("Patch subscription to run on infra nodes and wait for the operator pod to be ready")
			oc.Patch(t, env.GetOperatorNamespace(), "subscription", "servicemeshoperator", "merge", `
spec:
  config:
    nodeSelector:
      node-role.kubernetes.io/infra: ""
    tolerations:
    - effect: NoSchedule
      key: node-role.kubernetes.io/infra
      value: reserved
    - effect: NoExecute
      key: node-role.kubernetes.io/infra
      value: reserved
`)

			t.LogStepf("Verify operator pod is running on the infra node. Node expected: %s", workername)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				locator := pod.MatchingSelector("name=istio-operator", "openshift-operators")
				oc.WaitPodReady(t, locator)
				operatorPod := locator(t, oc.DefaultOC)
				shell.Execute(t,
					fmt.Sprintf(`oc get pod -n openshift-operators %s -o jsonpath='{.spec.nodeName}'`, operatorPod.Name),
					assert.OutputContains(
						workername,
						"Operator pod is running on the infra node",
						"Operator pod is not running on the infra node"))
			})
		})

		t.NewSubTest("control plane").Run(func(t test.TestHelper) {
			t.Log("Verify that all control plane pods are deployed on infra node when configured")
			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
			})

			DeployControlPlane(t)

			t.LogStep("Deploy SMCP and patch to run all control plane components on infra nodes")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  runtime:
    defaults:
      pod:
        nodeSelector:
          node-role.kubernetes.io/infra: ""
        tolerations:
        - effect: NoSchedule
          key: node-role.kubernetes.io/infra
          value: reserved
        - effect: NoExecute
          key: node-role.kubernetes.io/infra
          value: reserved
`)
				oc.WaitSMCPReady(t, meshNamespace, smcpName)
			})

			t.LogStep("Verify that the following control plane pods are running on the infra node: istiod, istio-ingressgateway, istio-egressgateway, jaeger, grafana, prometheus")
			istioPodLabelSelectors := []string{"app=istiod", "app=istio-ingressgateway", "app=istio-egressgateway", "app=grafana", "app=prometheus", "app=jaeger"}
			for _, pLabel := range istioPodLabelSelectors {
				assertPodScheduledToNode(t, pLabel)
			}

		})
	})
}

func assertPodScheduledToNode(t test.TestHelper, pLabel string) {
	// Increased the between retries to account for the time it takes for the pods to be scheduled on the infra node meanwhile other relocations are done at the same time.
	// Jaeger pod takes the longest to be scheduled on the infra node.
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(2*time.Second), func(t test.TestHelper) {
		podLocator := pod.MatchingSelector(pLabel, meshNamespace)
		po := podLocator(t, oc.DefaultOC)
		shell.Execute(t,
			fmt.Sprintf(`oc get pod -n %s %s -o jsonpath='{.spec.nodeName}'`, meshNamespace, po.Name),
			assert.OutputContains(
				workername,
				fmt.Sprintf("%s is running on the infra node", po.Name),
				fmt.Sprintf("%s is not running on the infra node", po.Name)))
	})
}

func pickWorkerNode(t test.TestHelper) string {
	workerNodes := shell.Execute(t, "oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[*].metadata.name}'")
	operatorNode := shell.Execute(t, "oc get pods -n openshift-operators -l name=istio-operator -o jsonpath='{.items[0].spec.nodeName}'")
	for _, node := range strings.Split(workerNodes, " ") {
		node = strings.TrimSpace(node)
		if node != operatorNode {
			return node
		}
	}
	t.Fatalf("could not find worker node")
	panic("we never get here because of the Fatalf call above")
}

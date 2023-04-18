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

	"golang.org/x/exp/slices"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var workername string

func TestDeployOnInfraNodes(t *testing.T) {
	NewTest(t).Id("T40").Groups(Full).Run(func(t TestHelper) {
		t.Log("This test verifies that the OSSM operator and Istio components can be configured to run on infrastructure nodes")
		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.T().Skip("Deploy On Infra node is available in SMCP versions v2.3+")
		}
		if Smcp.Rosa {
			t.Skip("Skipping test on ROSA due to lack of permissions")
		}
		t.Cleanup(func() {
			shell.Execute(t,
				`oc adm taint nodes -l node-role.kubernetes.io/infra node-role.kubernetes.io/infra=reserved:NoSchedule- node-role.kubernetes.io/infra=reserved:NoExecute-`)
			oc.Label(t, "", "node", workername, "node-role.kubernetes.io/infra-")
			oc.Label(t, "", "node", workername, "node-role.kubernetes.io-")
		})

		t.LogStep("Setup: Get a worker node from the cluster that does not have the istio operator installed, label it as infra")
		workername = pickWorkerNode(t)
		oc.Label(t, "", "node", workername, "node-role.kubernetes.io/infra=")
		oc.Label(t, "", "node", workername, "node-role.kubernetes.io=infra")
		t.Log(fmt.Sprintf("Worker node selected: %s", workername))

		t.LogStep("Taint node and edit the subscription to add the infra node to the node selector")
		shell.Execute(t,
			`oc adm taint nodes -l node-role.kubernetes.io/infra node-role.kubernetes.io/infra=reserved:NoSchedule node-role.kubernetes.io/infra=reserved:NoExecute`)

		t.NewSubTest("operator").Run(func(t TestHelper) {
			t.Log("Verify OSSM Operator is deployed on infra node when configured")
			t.Log("Reference: https://issues.redhat.com/browse/OSSM-2342")
			t.Cleanup(func() {
				shell.Execute(t,
					`oc patch subscription servicemeshoperator -n openshift-operators --type json -p='[{"op": "remove", "path": "/spec/config/tolerations"}]'`)
				oc.WaitSMCPReady(t, meshNamespace, smcpName) // Wait for the SMCP to be ready before move to next subtest case (Because untaint)
			})

			t.LogStep("Patch subscription to run on infra nodes and wait for the operator pod to be ready")
			oc.Patch(t, "openshift-operators", "subscription", "servicemeshoperator", "merge", `
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

			t.LogStep(fmt.Sprintf("Verify operator pod is running on the infra node. Node expected: %s", workername))
			retry.UntilSuccess(t, func(t test.TestHelper) {
				locator := pod.MatchingSelector("name=istio-operator", "openshift-operators")
				operatorPod := locator(t, oc.DefaultOC)
				oc.WaitPodReady(t, locator)
				shell.Execute(t,
					fmt.Sprintf(`oc get pod -n openshift-operators %s -o jsonpath='{.spec.nodeName}'`, operatorPod.Name),
					assert.OutputContains(
						workername,
						"Success: Operator pod is running on the infra node",
						"Error: Operator pod is not running on the infra node"))
			})
		})

		t.NewSubTest("control plane").Run(func(t TestHelper) {
			t.Log("Testing: Run OSSM Operator on infra nodes")
			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
				oc.ApplyTemplate(t, meshNamespace, GetSMCPTemplate(env.GetDefaultSMCPVersion()), Smcp)
				oc.WaitSMCPReady(t, meshNamespace, smcpName)
			})

			t.LogStep("Patch SMCP to run on infra nodes and wait for the SMCP to be ready")
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

			t.LogStep("Verify that the smcp pods are running on the infra node. Pod expected to be moved: istiod, istio-ingressgateway, istio-egressgateway, jaeger, grafana, prometheus")
			verifyPodRunningInNode(t)
		})
	})
}

func verifyPodRunningInNode(t TestHelper) {
	istioPods := []string{"istiod", "istio-ingressgateway", "istio-egressgateway", "jaeger", "grafana", "prometheus"}
	for _, p := range istioPods {
		podLocator := pod.MatchingSelector("app="+p, meshNamespace)
		po := podLocator(t, oc.DefaultOC)
		retry.UntilSuccess(t, func(t test.TestHelper) {
			nsPods := getAllPodsFromNode(t, meshNamespace)
			if !slices.Contains(nsPods, po.Name) {
				t.Log("The pod is not running on the infra node, delete it and wait for it to be recreated")
				oc.DeletePod(t, pod.MatchingSelector("app="+p, meshNamespace))
				nsPods = getAllPodsFromNode(t, meshNamespace)
				podLocator = pod.MatchingSelector("app="+p, meshNamespace)
				po = podLocator(t, oc.DefaultOC)
				if !slices.Contains(nsPods, po.Name) {
					t.Fatalf("Pod %s is not running on the infra node", p)
				}
			}
		})
	}
}

func pickWorkerNode(t test.TestHelper) string {
	workername := shell.Execute(t, "oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[0].metadata.name}'")
	actualNode := shell.Execute(t, `oc get pods -n openshift-operators -l name=istio-operator -o jsonpath='{.items[0].spec.nodeName}'`)
	if workername == actualNode {
		// If the worker node is the same as the node where the operator is running, pick the second worker node
		workername = shell.Execute(t, "oc get nodes -l node-role.kubernetes.io/worker= -o jsonpath='{.items[1].metadata.name}'")
	}

	return workername
}

func getAllPodsFromNode(t test.TestHelper, namespace string) []string {
	// split the output into a list of strings
	nsPodsOutput := shell.Executef(t,
		`kubectl get pods -n %s --field-selector spec.nodeName=%s -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}'`,
		namespace,
		workername)
	nsPods := strings.Split(nsPodsOutput, "\n")
	nonEmptyPods := []string{}
	for _, pod := range nsPods {
		if pod != "" {
			nonEmptyPods = append(nonEmptyPods, pod)
		}
	}

	return nonEmptyPods
}

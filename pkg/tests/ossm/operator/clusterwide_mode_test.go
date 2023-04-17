package operator

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestClusterWideMode(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {
		t.Log("This test verifies the behavior of SMCP.spec.mode: ClusterWide")

		meshNamespace := env.GetDefaultMeshNamespace()
		istiodDeployment := fmt.Sprintf("istiod-%s", ossm.Smcp.Name)

		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
			deleteMemberNamespaces(t, 50)
		})

		t.LogStepf("Delete and recreate namespace %s", meshNamespace)
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Install cluster-wide SMCP")
		oc.ApplyTemplate(t, meshNamespace, clusterWideSMCP, ossm.Smcp)

		t.LogStep("Wait for SMCP to be Ready")
		oc.WaitSMCPReady(t, meshNamespace, env.GetDefaultSMCPName())

		t.NewSubTest("SMMR auto-creation").Run(func(t test.TestHelper) {
			t.LogStep("Check whether SMMR is created automatically")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Get(t, meshNamespace, "servicemeshmemberroll", "default",
					assert.OutputContains("default",
						"The SMMR was created immediately after the SMCP was created",
						"The SMMR resource was not created"))
			})
		})

		t.NewSubTest("default namespace selector").Run(func(t test.TestHelper) {
			t.Log("Check whether namespaces with the label istio-injection=enabled become members automatically")

			t.LogStep("Create 50 member namespaces")
			createMemberNamespaces(t, 50)

			t.LogStep("Wait for SMMR to be Ready")
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check whether the SMMR shows all 50 namespaces as members")
			shell.Execute(t,
				fmt.Sprintf("oc -n %s get smmr default", meshNamespace),
				require.OutputContains("50/50",
					"all 50 namespaces are members",
					"expected SMMR to show 50 member namespaces, but that wasn't the case"))
		})

		t.NewSubTest("customize SMMR").Run(func(t test.TestHelper) {
			t.Log("Check whether the SMMR can be modified")

			t.LogStep("Configure static members member-0 and member-1 in SMMR")
			oc.ApplyString(t, meshNamespace, customSMMR)
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check whether the SMMR shows only two namespaces as members")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf("oc -n %s get smmr default", meshNamespace),
					require.OutputContains("2/2",
						"two namespaces are members",
						"expected SMMR to show 2 member namespaces, but that wasn't the case"))
			})

			t.LogStep("Reset member selector back to default")
			oc.ApplyString(t, meshNamespace, defaultSMMR)
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check whether the SMMR shows all 50 namespaces as members")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf("oc -n %s get smmr default", meshNamespace),
					require.OutputContains("50/50",
						"all 50 namespaces are members",
						"expected SMMR to show 50 member namespaces, but that wasn't the case"))
			})
		})

		t.NewSubTest("cluster-scoped watches in istiod").Run(func(t test.TestHelper) {
			t.Log("Check whether istiod watches API resources at the cluster scope")

			t.LogStep("Enable Kubernetes API request logging in istiod Deployment")
			t.Log("Patch istiod deployment to add the --logKubernetesApiRequests flag to pilot-discovery")
			oc.Patch(t,
				meshNamespace, "deployment", istiodDeployment, "json",
				`[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--logKubernetesApiRequests"}]`)

			t.Log("Wait for istiod deployment rollout to complete")
			oc.WaitDeploymentRolloutComplete(t, meshNamespace, istiodDeployment)

			t.LogStep("Check whether the number of API requests on istiod startup is in the expected range for cluster-wide mode")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Logs(t,
					pod.MatchingSelector("app=istiod", meshNamespace),
					"discovery",
					assertNumberOfAPIRequestsBetween(10, 100))
			})
		})
	})
}

func deleteMemberNamespaces(t test.TestHelper, count int) {
	oc.DeleteNamespace(t, util.GenerateStrings("member-", count)...)
}

func createMemberNamespaces(t test.TestHelper, count int) {
	namespaces := []string{}
	yaml := ""
	for i := 0; i < count; i++ {
		namespaces = append(namespaces, fmt.Sprintf("member-%d", i))
		yaml += fmt.Sprintf(`
apiVersion: v1
kind: Namespace
metadata:
  name: member-%d
  labels:
    istio-injection: enabled
---`, i)
	}

	t.Logf("Creating %d namespaces with the label 'istio-injection=enabled': %v", count, namespaces)
	oc.ApplyString(t, "", yaml)
}

func assertNumberOfAPIRequestsBetween(min, max int) common.CheckFunc {
	return func(t test.TestHelper, output string) {
		numberOfRequests := 0
		scanner := bufio.NewScanner(strings.NewReader(output))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Performing Kubernetes API request") {
				numberOfRequests++
			}
		}
		if numberOfRequests < min || numberOfRequests > max {
			t.Errorf("expected number of API requests to be between %d and %d, but the actual number was %d", min, max, numberOfRequests)
		} else {
			t.LogSuccessf("number of API requests (%d) is in range (%d - %d)", numberOfRequests, min, max)
		}
	}
}

const (
	clusterWideSMCP = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
spec:
  version: v2.4
  mode: ClusterWide
  general:
    logging:
      componentLevels:
        default: info
  tracing:
    type: Jaeger
    sampling: 10000
  policy:
    type: Istiod
  addons:
    grafana:
      enabled: true
    jaeger:
      install:
        storage:
          type: Memory
    kiali:
      enabled: true
    prometheus:
      enabled: true
  telemetry:
    type: Istiod
  {{ if .Rosa }} 
  security:
    identity:
      type: ThirdParty
  {{ end }}`

	customSMMR = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - member-0
  - member-1
  memberSelectors: []`

	defaultSMMR = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchLabels:
      istio-injection: enabled`
)

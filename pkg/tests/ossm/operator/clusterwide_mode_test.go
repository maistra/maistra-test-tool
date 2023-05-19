package operator

import (
	"bufio"
	"fmt"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
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
	test "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestClusterWideMode(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.Disconnected).MinVersion(version.SMCP_2_4).Run(func(t test.TestHelper) {
		t.Log("This test verifies the behavior of SMCP.spec.mode: ClusterWide")

		smcpName := env.GetDefaultSMCPName()
		meshNamespace := env.GetDefaultMeshNamespace()
		istiodDeployment := fmt.Sprintf("istiod-%s", smcpName)

		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
			deleteMemberNamespaces(t, 50)
		})

		t.LogStepf("Delete and recreate namespace %s", meshNamespace)
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Install cluster-wide SMCP")
		oc.ApplyTemplate(t, meshNamespace, clusterWideSMCP, ossm.DefaultSMCP())

		t.LogStep("Wait for SMCP to be Ready")
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

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

		t.NewSubTest("RoleBindings verification").Run(func(t test.TestHelper) {
			t.Log("Related to OSSM-3468")
			t.LogStep("Check that Rolebindings are not created in the member namespaces")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Get(t, "member-0", "rolebindings", "",
					assert.OutputContains("system:deployers",
						"The Rolebings contains system:deployers role",
						"The Rolebings does not contains system:deployers role"),
					assert.OutputContains("system:image-builders",
						"The Rolebings contains system:image-builders role",
						"The Rolebings does not contains system:image-builders role"),
					assert.OutputContains("system:image-pullers",
						"The Rolebings contains system:image-pullers role",
						"The Rolebings does not contains system:image-pullers role"),
					assert.OutputDoesNotContain("istiod-clusterrole-basic-istio-system",
						"The Rolebings does not contains istiod-clusterrole-basic-istio-system role",
						"The Rolebings contains istiod-clusterrole-basic-istio-system role"),
					assert.OutputDoesNotContain("istiod-gateway-controller-basic-istio-system",
						"The Rolebings does not contains istiod-gateway-controller-basic-istio-system role",
						"The Rolebings contains istiod-gateway-controller-basic-istio-system role"))
			})
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

			t.LogStep("Check the use of IN operator in member selector matchExpressions")
			oc.ApplyString(t, meshNamespace, smmrInOperator)
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check whether the SMMR shows only one namespace as members: member-0")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf("oc -n %s get smmr default", meshNamespace),
					require.OutputContains("1/1",
						"one namespace are in members",
						"expected SMMR to show 1 member namespace, but that wasn't the case"))
				shell.Execute(t,
					fmt.Sprintf("oc -n %s describe smmr default", meshNamespace),
					require.OutputContains("member-0",
						"member-0 is in members",
						"expected SMMR to show member-0 as member namespace, but that wasn't the case"))
			})

			t.LogStep("Check the use of multiple selector at the same time")
			oc.ApplyString(t, meshNamespace, smmrMultipleSelectors)
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check whether the SMMR shows only namespaces as members: member-0")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf("oc -n %s get smmr default", meshNamespace),
					require.OutputContains("1/1",
						"one namespace are in members",
						"expected SMMR to show 1 member namespace, but that wasn't the case"))
			})

			t.LogStep("Check the use of NotIn operator in member selector matchExpressions")
			oc.ApplyString(t, meshNamespace, smmrNotInOperator)
			oc.WaitSMMRReady(t, meshNamespace)

			t.LogStep("Check whether the SMMR shows all the namespaces except: member-0")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf("oc -n %s get smmr default", meshNamespace),
					require.OutputContains("55/55",
						"All the namespaces are in members except member-0",
						"expected SMMR to show 55 member namespace, but that wasn't the case"))
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

		t.NewSubTest("verify sidecar injection").Run(func(t test.TestHelper) {
			t.Log("Check if sidecar injeection works properly in clustewide mode")

			t.Cleanup(func() {
				app.Uninstall(t, app.Httpbin("member-0"))
			})

			t.LogStep("Install httpbin in member-0 namespace")
			app.InstallAndWaitReady(t,
				app.Httpbin("member-0"))

			t.LogStep("Verify that sidecar is injected in httpbin pod")
			shell.Execute(t,
				`oc -n member-0 get pods -l app=httpbin --no-headers`,
				assert.OutputContains(
					"2/2",
					"Side car injected in httpbin pod",
					"Expected 2 pods with sidecar injected, but that wasn't the case"))
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

		t.NewSubTest("cluster wide works with profiles").Run(func(t test.TestHelper) {
			t.Log("Check whether the cluster wide feature works with profiles")

			t.LogStep("Delete SMCP and SMMR")
			oc.RecreateNamespace(t, meshNamespace)

			t.LogStep("Create a profile with a cluster wide feature and restart OSSM operator")
			oc.CreateConfigMapFromFiles(t,
				"openshift-operators",
				"smcp-templates",
				ossm.GetProfileFile())
			podLocator := pod.MatchingSelector("name=istio-operator", "openshift-operators")
			oc.DeletePod(t, podLocator)
			oc.WaitPodReady(t, podLocator)

			t.LogStep("Deploy SMCP with the profile")
			oc.ApplyTemplate(t,
				meshNamespace,
				clusterWideSMCPWithProfile,
				map[string]string{"Name": "cluster-wide", "Version": env.GetSMCPVersion().String()})
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			t.LogStep("Check whether SMMR is created automatically")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Get(t, meshNamespace, "servicemeshmemberroll", "default",
					assert.OutputContains("default",
						"The SMMR was created immediately after the SMCP was created",
						"The SMMR resource was not created"))
			})
		})
	})
}

func deleteMemberNamespaces(t test.TestHelper, count int) {
	oc.DeleteNamespace(t, util.GenerateStrings("member-", count)...)
}

func createMemberNamespaces(t test.TestHelper, count int) {
	var namespaces []string
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
  version: {{ .Version }}
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

	clusterWideSMCPWithProfile = `
	apiVersion: maistra.io/v2
	kind: ServiceMeshControlPlane
	metadata:
	  name: {{ .Name }}
	spec:
	  version: {{ .Version }}
	  profiles:
	  - clusterScoped`

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

	smmrMultipleSelectors = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: In
      values:
      - member-0
    - key: kubernetes.io/metadata.name
      operator: NotIn
      values:
      - member-1`

	defaultSMMR = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchLabels:
      istio-injection: enabled`

	smmrInOperator = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: In
      values:
      - member-0`

	smmrNotInOperator = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchExpressions:
    - key: kubernetes.io/metadata.name
      operator: NotIn
      values:
      - member-0`
)

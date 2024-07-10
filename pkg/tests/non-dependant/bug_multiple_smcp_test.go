package non_dependant

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestSMCPMultiple(t *testing.T) {
	NewTest(t).Id("T36").Groups(Full, ARM).Run(func(t TestHelper) {
		t.Log("This test verifies whether the operator only reconciles one SMCP when two exist in a namespace")
		t.Log("See https://issues.redhat.com/browse/OSSM-2419")

		t.Cleanup(func() {
			t.LogStepf("Delete namespace %s", meshNamespace)
			oc.RecreateNamespace(t, meshNamespace)

			if env.GetOperatorVersion().LessThan(version.OPERATOR_2_6_0) {
				t.LogStep("Delete operator to recreate the ValidationWebhookConfiguration")
				oc.DeletePod(t, pod.MatchingSelector("name=istio-operator", env.GetOperatorNamespace()))
				t.LogStep("Wait for operator pod to be ready")
				oc.WaitPodReady(t, pod.MatchingSelector("name=istio-operator", env.GetOperatorNamespace()))
				t.LogStep("Check whether ValidatingWebhookConfiguration exists")
				retry.UntilSuccess(t, func(t TestHelper) {
					oc.Get(t, "", "validatingwebhookconfiguration", "openshift-operators.servicemesh-resources.maistra.io")
					t.LogSuccess("ValidatingWebhookConfiguration was recreated by the operator")
				})
			}
		})

		t.LogStepf("Delete and recreate namespace %s", meshNamespace)
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Create the first SMCP")
		smcp1 := ossm.DefaultSMCP().WithName("smcp1")
		ossm.InstallSMCPCustom(t, meshNamespace, smcp1)

		t.LogStep("Check whether the first SMCP gets reconciled and becomes ready")
		oc.WaitSMCPReady(t, meshNamespace, smcp1.Name)
		t.LogSuccess("First SMCP is ready")

		if env.GetOperatorVersion().GreaterThanOrEqual(version.OPERATOR_2_6_0) {
			t.LogStep("Check that the validationwebhook prevent of creating the second SMCP")
			shell.Execute(t,
				fmt.Sprintf("echo \"%s\" | oc create -f - -n %s || true", simpleSmcp(), meshNamespace),
				assert.OutputContains("admission webhook \"smcp.validation.maistra.io\" denied the request: only one service mesh may be installed per project/namespace",
					"Validationwebhook prevents of creating the second SMCP",
					"Expect that creation of second smcp fails on validationwebhook"))
		} else {
			t.LogStep("Delete the operator's ValidationWebhookConfiguration to be able to install the second SMCP")
			oc.DeleteResource(t, "", "validatingwebhookconfiguration", "openshift-operators.servicemesh-resources.maistra.io")
			t.LogStep("Create the second SMCP")
			smcp2 := ossm.DefaultSMCP().WithName("smcp2")
			ossm.InstallSMCPCustom(t, meshNamespace, smcp2)
			ossm.InstallSMCPCustom(t, meshNamespace, smcp2)

			ossm.InstallSMCPCustom(t, meshNamespace, smcp2)

			t.LogStep("Check whether the second SMCP shows ErrMultipleSMCPs")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Get(t, meshNamespace,
					"smcp", smcp2.Name,
					assert.OutputContains("ErrMultipleSMCPs",
						"The second SMCP status is ErrMultipleSMCPs",
						"The second SMCP status is not ErrMultipleSMCPs"))
			})
		}
	})
}

func simpleSmcp() string {
	return fmt.Sprintf(`
kind: ServiceMeshControlPlane
apiVersion: maistra.io/v2
metadata:
  name: basic
spec: 
  version: %s
`, env.GetSMCPVersion())
}

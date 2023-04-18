package non_dependant

import (
	_ "embed"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSMCPMultiple(t *testing.T) {
	NewTest(t).Id("T36").Groups(Full).Run(func(t TestHelper) {
		t.Log("This test verifies whether the operator only reconciles one SMCP when two exist in a namespace")
		t.Log("See https://issues.redhat.com/browse/OSSM-2189")

		smcp1 := ossm.SMCP{Name: "smcp1", Namespace: meshNamespace, Rosa: env.IsRosa()}
		smcp2 := ossm.SMCP{Name: "smcp2", Namespace: meshNamespace, Rosa: env.IsRosa()}

		t.Cleanup(func() {
			t.LogStepf("Delete namespace %s", meshNamespace)
			oc.RecreateNamespace(t, meshNamespace)

			t.LogStep("Delete operator to recreate the ValidationWebhookConfiguration")
			shell.Execute(t, "oc -n openshift-operators delete pod -l name=istio-operator")

			t.LogStep("Check whether ValidatingWebhookConfiguration exists")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Get(t, "", "validatingwebhookconfiguration", "openshift-operators.servicemesh-resources.maistra.io")
				t.LogSuccess("ValidatingWebhookConfiguration was recreated by the operator")
			})
		})

		t.LogStepf("Delete and recreate namespace %s", meshNamespace)
		oc.RecreateNamespace(t, meshNamespace)

		t.LogStep("Delete the operator's ValidationWebhookConfiguration")
		oc.DeleteResource(t, "", "validatingwebhookconfiguration", "openshift-operators.servicemesh-resources.maistra.io")

		t.LogStep("Create the first SMCP")
		oc.ApplyTemplate(t, meshNamespace, ossm.GetDefaultSMCPTemplate(), smcp1)

		t.LogStep("Check whether the first SMCP gets reconciled and becomes ready")
		oc.WaitSMCPReady(t, meshNamespace, smcp1.Name)
		t.LogSuccess("First SMCP is ready")

		t.LogStep("Create the second SMCP")
		oc.ApplyTemplate(t, meshNamespace, ossm.GetDefaultSMCPTemplate(), smcp2)

		t.LogStep("Check whether the second SMCP shows ErrMultipleSMCPs")
		retry.UntilSuccess(t, func(t TestHelper) {
			oc.Get(t, meshNamespace,
				"smcp", smcp2.Name,
				assert.OutputContains("ErrMultipleSMCPs",
					"The second SMCP status is ErrMultipleSMCPs",
					"The second SMCP status is not ErrMultipleSMCPs"))
		})
	})
}

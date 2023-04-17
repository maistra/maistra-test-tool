package ossm

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

//go:embed yaml/smcp_meta_v2.3.yaml
var smcpV23_template_meta string

//go:embed yaml/smcp_v2.3.yaml
var smcpV23_template_1 string

//go:embed yaml/smmr.yaml
var SMMR string

// TestSMCPMutiple tests If multiple SMCPs exist in a namespace, the controller reconciles them all. Jira ticket: https://issues.redhat.com/browse/OSSM-2434

func TestSMCPMutiple(t *testing.T) {
	NewTest(t).Id("T36").Groups(Full).Run(func(t TestHelper) {

		ns := "multiple"
		t.Cleanup(func() {
			shell.Execute(t, fmt.Sprintf(`oc -n openshift-operators delete pod -l name=istio-operator`),
				assert.OutputContains("deleted",
					"Expected to istio-operator restarted",
					"Expected istio-operator not restarted",
				))

			oc.RecreateNamespace(t, ns)
		})

		t.LogStep("Delete the Validation webhook")
		oc.CreateNamespace(t, ns)

		shell.Execute(t,
			fmt.Sprintf(`oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io`),
			assert.OutputContains("deleted",
				"Delete the Validation webhood",
				"Validation webhook not deleted"))

		t.LogStep("Create the first SMCP in multiple ns")
		oc.ApplyTemplate(t, ns, smcpV23_template_1, Smcp)

		t.LogStep("Create the second SMCP in multiple ns")
		oc.ApplyString(t, ns, smcpV23_template_meta)

		t.LogStep("Validate the multiple SMCP status in multiple ns")

		oc.WaitSMCPReady(t, ns, "basic")

		shell.Execute(t,
			fmt.Sprintf(`oc get -n %s smcp/meta -o wide`, ns),
			assert.OutputContains("ErrMultipleSMCPs",
				"Verfied the second SMCP",
				"Second SMCP is not verified"))

	})

}

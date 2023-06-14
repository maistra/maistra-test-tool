package operator

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestOperatorCanReconcileSMCPWhenIstiodOffline(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.Disconnected).Run(func(t test.TestHelper) {
		t.Log("This test checks if the operator can reconcile an SMCP even if the istiod pod is missing")
		t.Log("See https://issues.redhat.com/browse/OSSM-3235")

		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			t.Skip("Skipped because OSSM-3235 is only fixed in 2.4+")
		}

		meshNamespace := env.GetDefaultMeshNamespace()
		smcpName := env.GetDefaultSMCPName()
		istiodDeployment := fmt.Sprintf("istiod-%s", smcpName)

		t.Cleanup(func() {
			oc.ScaleDeploymentAndWait(t, meshNamespace, istiodDeployment, 1)
		})

		t.LogStep("Install SMCP and wait for it to be Ready")
		ossm.InstallSMCP(t, meshNamespace)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Scale istiod to zero replicas, so that the validation webhook goes offline")
		oc.ScaleDeploymentAndWait(t, meshNamespace, istiodDeployment, 0)

		t.LogStep("Force SMCP to be reconciled")
		oc.TouchSMCP(t, meshNamespace, smcpName)

		t.LogStep("Wait for SMCP to be ready; if this doesn't happen, the ValidationWebhookConfiguration is probably missing the correct objectSelector")
		oc.WaitSMCPReady(t, meshNamespace, smcpName)
	})
}

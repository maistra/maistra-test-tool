package observability

import (
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

const customPrometheusNs = "custom-prometheus"

// TestCustomPrometheusOperator tests integration with OSE Prometheus Operator that is a member of the mesh and collects
// application metrics over mTLS.
func TestCustomPrometheusOperator(t *testing.T) {
	test.NewTest(t).Id("custom-prometheus-operator").Groups(test.Full).Run(func(t test.TestHelper) {
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("integration with OpenShift Monitoring stack is not supported in OSSM older than v2.4.0")
		}

		t.Cleanup(func() {
			oc.DeleteNamespace(t, customPrometheusNs)
		})

		//meshValues := map[string]string{
		//	"Name":    smcpName,
		//	"Version": smcpVer.String(),
		//	"Member":  ns.Foo,
		//}

		oc.CreateNamespace(t, customPrometheusNs)
	})
}

package observability

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/prometheus"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestFederatedOpenShiftMonitoring requires OpenShift Monitoring stack to be enabled.
// See the comment on TestOpenShiftMonitoring for help setting up on crc
func TestFederatedOpenShiftMonitoring(t *testing.T) {
	test.NewTest(t).Id("federated-openshift-monitoring-integration").Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
		meshValues := map[string]interface{}{
			"Name":    smcpName,
			"Version": env.GetSMCPVersion().String(),
			"Member":  ns.Foo,
			"Rosa":    env.IsRosa(),
		}

		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, monitoringNs, clusterMonitoringConfigTmpl, map[string]bool{"Enabled": false})
			oc.RecreateNamespace(t, ns.Foo)
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStep("Waiting until user workload monitoring stack is up and running")
		oc.ApplyTemplate(t, monitoringNs, clusterMonitoringConfigTmpl, map[string]bool{"Enabled": true})
		oc.WaitPodsExist(t, userWorkloadMonitoringNs)
		oc.WaitAllPodsReady(t, userWorkloadMonitoringNs)

		t.LogStep("Deploying SMCP")
		oc.ApplyTemplate(t, meshNamespace, federatedMonitoringMeshTmpl, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Deploy httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
		oc.WaitPodsExist(t, ns.Foo)
		oc.WaitAllPodsReady(t, ns.Foo)

		t.LogStep("Generate some ingress traffic")
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.6/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Apply federated metrics monitor")
		oc.ApplyString(t, meshNamespace, federatedMonitor)

		t.LogStep("Check istiod metrics")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			resp := prometheus.ThanosQuery(t, monitoringNs, `pilot_info{mesh_id="unique-mesh-id"}`)
			if len(resp.Data.Result) == 0 {
				t.Errorf("No data points received from Prometheus API. Response status: %s", resp.Status)
			}
		})

		t.LogStep("Check httpbin metrics")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			resp := prometheus.ThanosQuery(t, monitoringNs, `istio_requests_total{mesh_id="unique-mesh-id"}`)
			if len(resp.Data.Result) == 0 {
				t.Errorf("No data points received from Prometheus API. Response status: %s", resp.Status)
			}
		})
	})
}

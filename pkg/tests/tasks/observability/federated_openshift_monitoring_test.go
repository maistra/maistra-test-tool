// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observability

import (
	"fmt"
	"net/http"
	"strings"
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

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestFederatedOpenShiftMonitoring requires OpenShift Monitoring stack to be enabled.
// See the comment on TestOpenShiftMonitoring for help setting up on crc
func TestFederatedOpenShiftMonitoring(t *testing.T) {
	NewTest(t).Id("federated-openshift-monitoring-integration").Groups(Full, ARM, Disconnected).Run(func(t TestHelper) {
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

		t.LogStep("Apply federated metrics monitor")
		oc.ApplyString(t, meshNamespace, federatedMonitor)

		t.LogStep("Wait until istio targets appear in the Prometheus")
		retry.UntilSuccess(t, func(t TestHelper) {
			resp := prometheus.ThanosTargets(t, monitoringNs)
			if !strings.Contains(resp, "serviceMonitor/istio-system/istio-federation") {
				t.Error("Istio Prometheus target serviceMonitor/istio-system/istio-federation are not ready")
			}
		})

		t.LogStep("Generate some ingress traffic")
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.6/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Check istiod metrics")
		retry.UntilSuccess(t, func(t TestHelper) {
			resp := prometheus.ThanosQuery(t, monitoringNs, `pilot_info{mesh_id="unique-mesh-id"}`)
			if len(resp.Data.Result) == 0 {
				t.Errorf("No data points received from Prometheus API. Response status: %s", resp.Status)
			}
		})

		t.LogStep("Check httpbin metrics")
		retry.UntilSuccess(t, func(t TestHelper) {
			resp := prometheus.ThanosQuery(t, monitoringNs, `istio_requests_total{mesh_id="unique-mesh-id"}`)
			if len(resp.Data.Result) == 0 {
				t.Errorf("No data points received from Prometheus API. Response status: %s", resp.Status)
			}
		})
	})
}

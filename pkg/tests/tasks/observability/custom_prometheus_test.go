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
	"github.com/maistra/maistra-test-tool/pkg/prometheusoperator"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/prometheus"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestCustomPrometheus(t *testing.T) {
	const customPrometheusNs = "custom-prometheus-operator"

	test.NewTest(t).Id("custom-prometheus").Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("Extension providers are not supported in OSSM older than v2.4.0")
		}
		ocpVersion := version.ParseVersion(oc.GetOCPVersion(t))
		if ocpVersion.LessThan(version.OCP_4_11) {
			t.Skip("Custom Prometheus operator from red hat is not supported in OCP older than v4.11")
		}

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
			oc.RecreateNamespace(t, meshNamespace)
			prometheusoperator.Uninstall(t)
		})

		t.LogStep("Installing Prometheus operator")
		prometheusoperator.Install(t)

		t.LogStep("Creating SMCP with Prometheus extension provider")
		createSmcpWithPrometheusExtensionProvider(t, meshNamespace, customPrometheusNs, ns.Bookinfo)

		t.LogStep("Installing custom Prometheus")
		prometheusoperator.InstalPrometheusInstance(t, meshNamespace, ns.Bookinfo)

		t.LogStep("Intalling Bookinfo app")
		oc.WaitSMCPReady(t, meshNamespace, "basic")
		bookinfoApp := app.Bookinfo(ns.Bookinfo)
		bookinfoApp.Install(t)

		t.LogStep("Enabling telemetry")
		enablePrometheusTelemetry(t, meshNamespace)

		t.LogStep("Enabling monitoring")
		enableIstiodMonitoring(t, customPrometheusNs, meshNamespace)
		enableIstioProxiesMonitoring(t, customPrometheusNs, meshNamespace, ns.Bookinfo)
		enableAppMtlsMonitoring(t, customPrometheusNs, ns.Bookinfo)

		t.LogStep("Waiting for installs to complete")
		bookinfoApp.WaitReady(t)

		t.LogStep("Sending request to Bookinfo app")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, app.BookinfoProductPageURL(t, meshNamespace), nil, assert.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Testing if telemetry was enabled")
		ocWaitJsonpath(t, meshNamespace, "smcp", "basic",
			"{.status.appliedValues.istio.telemetry.enabled}", "true",
			"Telemetry was enabled.", "Telemetry failed to enable.")

		t.LogStep("Testing if 'istio_requests_total' metric is available through Prometheus API")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			resp := prometheus.CustomPrometheusQuery(t, customPrometheusNs,
				fmt.Sprintf(`istio_requests_total{namespace="%s",container="istio-proxy",source_app="istio-ingressgateway",destination_app="productpage"}`, ns.Bookinfo))

			if len(resp.Data.Result) == 0 {
				t.Errorf("No data points received from Prometheus API, status: %s", resp.Status)
			}
		})
	})
}

// utility functions to consistently escape shell arguments for external commands
func shellArg(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, `'`, `'\''`))
}
func shellArgf(format string, a ...any) string {
	return shellArg(fmt.Sprintf(format, a...))
}

func ocWaitJsonpath(t test.TestHelper, ns, kind, name, jsonpath, expected, successMessage, failureMsg string) {
	t.T().Helper()
	timeout := "1s"
	cmd := fmt.Sprintf(`oc -n %s wait %s --for %s --timeout %s`,
		shellArg(ns), shellArgf("%s/%s", kind, name), shellArgf("jsonpath=%s=%s", jsonpath, expected), timeout)
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.DefaultOC.Invoke(t, cmd, assert.OutputContains(" condition met\n", successMessage, failureMsg))
	})
}

func createSmcpWithPrometheusExtensionProvider(t test.TestHelper, smcpNs, prometheusNs, additionalSmmrNs string) {
	t.T().Helper()
	oc.ApplyTemplate(t, smcpNs, `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: basic
spec:
  addons:
    grafana:
      enabled: false
    kiali:
      enabled: false
    prometheus:
      enabled: false
  extensionProviders:
  - name: prometheus
    prometheus: {}
  gateways:
    egress:
      enabled: false
    openshiftRoute:
      enabled: false
  proxy:
    accessLogging:
      file:
        name: /dev/stdout
  security:
    dataPlane:
      mtls: true
    {{ if .Rosa }}
    identity:
      type: ThirdParty
    {{ end }}
  tracing:
    type: None`, map[string]interface{}{"Rosa": env.IsRosa()})

	oc.ApplyString(t, smcpNs, fmt.Sprintf(`
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - %s
  - %s`,
		prometheusNs,
		additionalSmmrNs))
}

func enablePrometheusTelemetry(t test.TestHelper, smcpNs string) {
	t.T().Helper()
	oc.ApplyString(t, smcpNs, `
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: enable-prometheus-metrics
spec:
  metrics:
  - providers:
    - name: prometheus
`)
}

func enableIstiodMonitoring(t test.TestHelper, prometheusNs, smcpNs string) {
	t.T().Helper()
	oc.ApplyString(t, prometheusNs,
		fmt.Sprintf(`
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: istiod-monitor
spec:
  targetLabels:
  - app
  selector:
    matchLabels:
      istio: pilot
  namespaceSelector:
    matchNames:
      - %s
  endpoints:
  - port: http-monitoring
    interval: 15s
    path: /metrics`,
			smcpNs))
}

func enableIstioProxiesMonitoring(t test.TestHelper, prometheusNs, smcpNs, additionalNs string) {
	t.T().Helper()
	oc.ApplyString(t, prometheusNs,
		fmt.Sprintf(`
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: istio-proxies-monitor
spec:
  selector:
    matchExpressions:
    - key: istio-prometheus-ignore
      operator: DoesNotExist
  namespaceSelector:
    matchNames:
    - %s
    - %s
  podMetricsEndpoints:
  - path: /stats/prometheus
    interval: 15s
    relabelings:
    - action: keep
      sourceLabels: [ __meta_kubernetes_pod_container_name ]
      regex: "istio-proxy"
    - action: keep
      sourceLabels: [ __meta_kubernetes_pod_annotationpresent_prometheus_io_scrape ]
    - action: replace
      regex: (\d+);(([A-Fa-f0-9]{1,4}::?){1,7}[A-Fa-f0-9]{1,4})
      replacement: '[$2]:$1'
      sourceLabels: [__meta_kubernetes_pod_annotation_prometheus_io_port, __meta_kubernetes_pod_ip]
      targetLabel: __address__
    - action: replace
      regex: (\d+);((([0-9]+?)(\.|$)){4})
      replacement: $2:$1
      sourceLabels: [__meta_kubernetes_pod_annotation_prometheus_io_port, __meta_kubernetes_pod_ip]
      targetLabel: __address__
    - action: labeldrop
      regex: "__meta_kubernetes_pod_label_(.+)"
    - sourceLabels: [ __meta_kubernetes_namespace ]
      action: replace
      targetLabel: namespace
    - sourceLabels: [ __meta_kubernetes_pod_name ]
      action: replace
      targetLabel: pod_name
    - action: replace
      replacement: "basic_%s"
      targetLabel: mesh_id`,
			smcpNs,
			additionalNs,
			smcpNs))
}

func enableAppMtlsMonitoring(t test.TestHelper, prometheusNs, bookinfoNs string) {
	t.T().Helper()
	oc.ApplyString(t, prometheusNs,
		fmt.Sprintf(`
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: app-metrics-monitor-mtls
spec:
  targetLabels:
  - app
  selector:
    matchLabels:
      app: productpage
  namespaceSelector:
    matchNames:
    - %s
  endpoints:
  - port: http
    path: /metrics
    interval: 15s
    scheme: https
    tlsConfig:
      caFile: /etc/prom-certs/root-cert.pem
      certFile: /etc/prom-certs/cert-chain.pem
      keyFile: /etc/prom-certs/key.pem
      insecureSkipVerify: true`,
			bookinfoNs))
}

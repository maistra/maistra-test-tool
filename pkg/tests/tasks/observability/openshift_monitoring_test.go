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
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

const (
	monitoringNs             = "openshift-monitoring"
	userWorkloadMonitoringNs = "openshift-user-workload-monitoring"
)

// TestOpenShiftMonitoring requires OpenShift Monitoring stack to be enabled.
// In case of running tests on CRC, you must enable the following setting:
// crc config set enable-cluster-monitoring true
// Make sure that you have sufficient resources to run monitoring stack - 8 CPU and 20 Gb of RAM should be enough:
// crc config set memory 20480
// crc config set cpus 8
func TestOpenShiftMonitoring(t *testing.T) {
	test.NewTest(t).Id("openshift-monitoring-integration").Groups(test.Full).Run(func(t test.TestHelper) {
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("integration with OpenShift Monitoring stack is not supported in OSSM older than v2.4.0")
		}
		smcpValues := map[string]string{
			"Name":    smcpName,
			"Version": smcpVer.String(),
			"Member":  ns.Foo,
		}

		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, monitoringNs, clusterMonitoringConfig, map[string]bool{"Enabled": false})
			oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
			oc.DeleteFromTemplate(t, meshNamespace, mesh, smcpValues)
			app.Uninstall(t, app.Httpbin(ns.Foo))
		})

		t.LogStep("Waiting until user workload monitoring stack is up and running")
		oc.ApplyTemplate(t, monitoringNs, clusterMonitoringConfig, map[string]bool{"Enabled": true})
		oc.WaitPodsExist(t, userWorkloadMonitoringNs)
		oc.WaitAllPodsReady(t, userWorkloadMonitoringNs)

		t.LogStep("Deploying SMCP")
		oc.ApplyTemplate(t, meshNamespace, mesh, smcpValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Enable Prometheus telemetry")
		oc.ApplyString(t, meshNamespace, enableTrafficMetrics)

		t.LogStep("Deploy httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
		oc.WaitPodsExist(t, ns.Foo)
		oc.WaitAllPodsReady(t, ns.Foo)

		t.LogStep("Apply Prometheus monitors")
		oc.ApplyString(t, meshNamespace, istiodServiceMonitor, istioProxyMonitor)
		oc.ApplyString(t, ns.Foo, istioProxyMonitor)

		t.LogStep("Generate some ingress traffic")
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Fetch Thanos secret")
		var secret string
		retry.UntilSuccess(t, func(t test.TestHelper) {
			secret = shell.Executef(t,
				`kubectl get secrets --no-headers -o custom-columns=":metadata.name" -n %s | grep prometheus-user-workload-token-`, userWorkloadMonitoringNs)
			if strings.Contains(secret, "Error") {
				t.Errorf("unexpected secret name: %s", secret)
			} else {
				secret = strings.TrimSuffix(secret, "\n")
			}
		})

		t.LogStep("Fetch Thanos token")
		var thanosToken string
		retry.UntilSuccess(t, func(t test.TestHelper) {
			thanosToken = shell.Executef(t, "oc get secret %s -n %s --template={{.data.token}} | base64 -d", secret, userWorkloadMonitoringNs)
			if strings.Contains(thanosToken, "Error") {
				t.Errorf("unexpected error: %s", thanosToken)
			}
		})

		t.LogStep("Check istiod metrics")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app.kubernetes.io/instance=thanos-querier", monitoringNs),
				"thanos-query",
				fmt.Sprintf(`curl -X GET -kG "https://localhost:9092/api/v1/query?namespace=%s&query=pilot_info"`+
					` --data-urlencode "query=up" -H "Authorization: Bearer %s"`, meshNamespace, thanosToken),
				assert.OutputContains(`"result":[{"metric":{"__name__":"pilot_info"`,
					"Successfully fetched pilot_info metrics", "Did not find pilot_info metric"),
			)
		})

		t.LogStep("Check httpbin metrics")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app.kubernetes.io/instance=thanos-querier", monitoringNs),
				"thanos-query",
				fmt.Sprintf(`curl -X GET -kG "https://localhost:9092/api/v1/query?namespace=%s&query=istio_requests_total"`+
					` --data-urlencode "query=up" -H "Authorization: Bearer %s"`, ns.Foo, thanosToken),
				assert.OutputContains(`"result":[{"metric":{"__name__":"istio_requests_total"`,
					"Successfully fetched istio_requests_total metrics", "Did not find istio_requests_total metric"),
			)
		})
	})
}

const (
	clusterMonitoringConfig = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-monitoring-config
data:
  config.yaml: |
    enableUserWorkload: {{ .Enabled }}
`
	mesh = `
apiVersion: maistra.io/v2
kind: ServiceMeshControlPlane
metadata:
  name: {{ .Name }}
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
  security:
    dataPlane:
      mtls: true
    manageNetworkPolicy: false
  tracing:
    type: None
  version: {{ .Version }}
---
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - {{ .Member }}
`
	enableTrafficMetrics = `
apiVersion: telemetry.istio.io/v1alpha1
kind: Telemetry
metadata:
  name: enable-prometheus-metrics
spec:
  metrics:
  - providers:
    - name: prometheus
`
	istiodServiceMonitor = `
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
  endpoints:
  - port: http-monitoring
    interval: 15s
`
	istioProxyMonitor = `
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: istio-proxies-monitor
spec:
  selector:
    matchExpressions:
    - key: istio-prometheus-ignore
      operator: DoesNotExist
  podMetricsEndpoints:
  - path: /stats/prometheus
    interval: 15s
    relabelings:
    - action: keep
      sourceLabels: [__meta_kubernetes_pod_container_name]
      regex: "istio-proxy"
    - action: keep
      sourceLabels: [__meta_kubernetes_pod_annotationpresent_prometheus_io_scrape]
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
    - action: replace
      regex: .*[revision]\":\"([^\"]+).*
      replacement: $1
      sourceLabels: [__meta_kubernetes_pod_annotation_sidecar_istio_io_status]
      targetLabel: revision
    - action: labeldrop
      regex: "__meta_kubernetes_pod_label_(.+)"
    - sourceLabels: [__meta_kubernetes_namespace]
      action: replace
      targetLabel: namespace
    - sourceLabels: [__meta_kubernetes_pod_name]
      action: replace
      targetLabel: pod_name
    - action: replace
      replacement: "eu-west-1"
      targetLabel: mesh_id
`
)

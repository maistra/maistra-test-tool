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
	kialiName = "kiali-user-workload-monitoring"
)

// TestOpenShiftMonitoring requires OpenShift Monitoring stack to be enabled.
// In case of running tests on CRC, you must enable the following setting:
// crc config set enable-cluster-monitoring true
// Make sure that you have sufficient resources to run monitoring stack - 8 CPU and 20 Gb of RAM should be enough:
// crc config set memory 20480
// crc config set cpus 8
func TestOpenShiftMonitoring(t *testing.T) {
	test.NewTest(t).Id("openshift-monitoring-integration").Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("integration with OpenShift Monitoring stack is not supported in OSSM older than v2.4.0")
		}
		meshValues := map[string]string{
			"Name":                smcpName,
			"Version":             smcpVer.String(),
			"Member":              ns.Foo,
			"manageNetworkPolicy": "false",
		}
		kialiValues := map[string]string{
			"SmcpName":      smcpName,
			"SmcpNamespace": meshNamespace,
		}

		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, monitoringNs, clusterMonitoringConfigTmpl, map[string]bool{"Enabled": false})
			oc.DeleteFromTemplate(t, meshNamespace, kialiUserWorkloadMonitoringTmpl, kialiValues)
		})

		t.LogStep("Waiting until user workload monitoring stack is up and running")
		oc.ApplyTemplate(t, monitoringNs, clusterMonitoringConfigTmpl, map[string]bool{"Enabled": true})
		oc.WaitPodsExist(t, userWorkloadMonitoringNs)
		oc.WaitAllPodsReady(t, userWorkloadMonitoringNs)

		t.LogStep("Deploying Kiali")
		oc.ApplyTemplate(t, meshNamespace, kialiUserWorkloadMonitoringTmpl, kialiValues)

		t.LogStep("Grant cluster-monitoring-view to Kiali")
		oc.ApplyTemplate(t, meshNamespace, kialiClusterMonitoringView, kialiValues)

		t.NewSubTest("SMCP manageNetworkPolicy false").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, istiodMonitor)
				oc.DeleteFromString(t, ns.Foo, istioProxyMonitor)
				oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
				app.Uninstall(t, app.Httpbin(ns.Foo))
				oc.DeleteFromTemplate(t, meshNamespace, meshTmpl, meshValues)
			})

			t.LogStep("Deploying SMCP")
			oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			waitKialiAndVerifyIsReconciled(t)

			t.LogStep("Fetch Kiali token")
			kialiToken := fetchKialiToken(t)

			t.LogStep("Enable Prometheus telemetry")
			oc.ApplyString(t, meshNamespace, enableTrafficMetrics)

			t.LogStep("Deploy httpbin")
			app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
			oc.WaitPodsExist(t, ns.Foo)
			oc.WaitAllPodsReady(t, ns.Foo)

			t.LogStep("Apply Prometheus monitors")
			oc.ApplyString(t, meshNamespace, istiodMonitor)
			oc.ApplyString(t, ns.Foo, istioProxyMonitor)

			generateTrafficAndcheckMetrics(t, kialiToken)
		})

		t.NewSubTest("SMCP manageNetworkPolicy true").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, istiodMonitor)
				oc.DeleteFromString(t, ns.Foo, istioProxyMonitor)
				oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
				app.Uninstall(t, app.Httpbin(ns.Foo))
				oc.DeleteFromTemplate(t, meshNamespace, networkPolicy, map[string]string{"namespace": meshNamespace})
				oc.DeleteFromTemplate(t, meshNamespace, meshTmpl, meshValues)
			})

			t.LogStep("Deploying SMCP")
			meshValues["manageNetworkPolicy"] = "true"
			oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			waitKialiAndVerifyIsReconciled(t)

			t.LogStep("Fetch Kiali token")
			kialiToken := fetchKialiToken(t)

			t.LogStep("Enable Prometheus telemetry")
			oc.ApplyString(t, meshNamespace, enableTrafficMetrics)

			t.LogStep("Deploy httpbin")
			app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
			oc.WaitPodsExist(t, ns.Foo)
			oc.WaitAllPodsReady(t, ns.Foo)

			t.LogStep("Deploying NetworkPolicy")
			oc.ApplyTemplate(t, meshNamespace, networkPolicy, map[string]string{"namespace": meshNamespace})
			oc.ApplyTemplate(t, ns.Foo, networkPolicy, map[string]string{"namespace": ns.Foo})

			t.LogStep("Apply Prometheus monitors")
			oc.ApplyString(t, meshNamespace, istiodMonitor)
			oc.ApplyString(t, ns.Foo, istioProxyMonitor)

			generateTrafficAndcheckMetrics(t, kialiToken)
		})
	})
}

func waitKialiAndVerifyIsReconciled(t test.TestHelper) {
	t.LogStep("Wait until Kiali is ready")
	oc.WaitCondition(t, meshNamespace, "Kiali", kialiName, "Successful")

	t.LogStep("Verify that Kiali was reconciled by Istio Operator")
	retry.UntilSuccess(t, func(t test.TestHelper) {
		accessibleNamespaces := shell.Executef(t, "oc get kiali %s -n %s -o jsonpath='{.spec.deployment.accessible_namespaces}'", kialiName, meshNamespace)
		if accessibleNamespaces != fmt.Sprintf(`["%s"]`, ns.Foo) {
			t.Errorf(`unexpected accessible namespaces: got '%s', expected: '["%s"]'`, accessibleNamespaces, ns.Foo)
		}
		configMapName := shell.Executef(t, "oc get kiali %s -n %s -o jsonpath='{.spec.external_services.istio.config_map_name}'", kialiName, meshNamespace)
		if configMapName != fmt.Sprintf("istio-%s", smcpName) {
			t.Errorf("unexpected istio config map name: got '%s', expected: 'istio-%s'", configMapName, smcpName)
		}
		sidecarInjectorConfigMapName := shell.Executef(t, "oc get kiali %s -n %s -o jsonpath='{.spec.external_services.istio.istio_sidecar_injector_config_map_name}'", kialiName, meshNamespace)
		if sidecarInjectorConfigMapName != fmt.Sprintf("istio-sidecar-injector-%s", smcpName) {
			t.Errorf("unexpected sidecar injecto config map name: got '%s', expected: 'istio-sidecar-injector-%s'", sidecarInjectorConfigMapName, smcpName)
		}
		deploymentName := shell.Executef(t, "oc get kiali %s -n %s -o jsonpath='{.spec.external_services.istio.istiod_deployment_name}'", kialiName, meshNamespace)
		if deploymentName != fmt.Sprintf("istiod-%s", smcpName) {
			t.Errorf("unexpected istiod deployment name: got '%s', expected: 'istiod-%s'", deploymentName, smcpName)
		}
		urlServiceVersion := shell.Executef(t, "oc get kiali %s -n %s -o jsonpath='{.spec.external_services.istio.url_service_version}'", kialiName, meshNamespace)
		if urlServiceVersion != fmt.Sprintf("http://istiod-%s.%s:15014/version", smcpName, meshNamespace) {
			t.Errorf("unexpected URL service version: got '%s', expected: 'http://istiod-%s.%s:15014/version'", urlServiceVersion, smcpName, meshNamespace)
		}
	})
}

func generateTrafficAndcheckMetrics(t test.TestHelper, thanosToken string) {
	t.LogStep("Generate some ingress traffic")
	oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.5/samples/httpbin/httpbin-gateway.yaml")
	httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
	retry.UntilSuccess(t, func(t test.TestHelper) {
		curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
	})

	t.LogStep("Check istiod metrics")
	checkMetricExists(t, meshNamespace, "pilot_info", thanosToken)

	t.LogStep("Check httpbin metrics")
	checkMetricExists(t, ns.Foo, "istio_requests_total", thanosToken)
}

func checkMetricExists(t test.TestHelper, ns, metricName, token string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelectorFirst("app.kubernetes.io/instance=thanos-querier", monitoringNs),
			"thanos-query",
			prometheusQuery(ns, metricName, token),
			assert.OutputContains(
				fmt.Sprintf(`"result":[{"metric":{"__name__":"%s"`, metricName),
				fmt.Sprintf("Successfully fetched %s metrics", metricName),
				fmt.Sprintf("Did not find %s metric", metricName)),
		)
	})
}

func prometheusQuery(ns, metricName, token string) string {
	return fmt.Sprintf(
		`curl -X GET -kG "https://localhost:9091/api/v1/query?namespace=%s&query=%s" --data-urlencode "query=up" -H "Authorization: Bearer %s"`,
		ns, metricName, token)
}

func fetchKialiToken(t test.TestHelper) string {
	var kialiToken string
	retry.UntilSuccess(t, func(t test.TestHelper) {
		kialiToken = shell.Executef(t, "oc exec -n %s $(oc get pods -n %s -l app=kiali -o jsonpath='{.items[].metadata.name}') "+
			"-- cat /var/run/secrets/kubernetes.io/serviceaccount/token", meshNamespace, meshNamespace)
		if strings.Contains(kialiToken, "Error") {
			t.Errorf("unexpected error: %s", kialiToken)
		}
	})
	return kialiToken
}

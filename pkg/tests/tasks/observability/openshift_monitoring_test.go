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
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
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
	NewTest(t).Id("openshift-monitoring-integration").Groups(Full, ARM, Disconnected).Run(func(t TestHelper) {
		smcpVer := env.GetSMCPVersion()
		if smcpVer.LessThan(version.SMCP_2_4) {
			t.Skip("integration with OpenShift Monitoring stack is not supported in OSSM older than v2.4.0")
		}
		meshValues := map[string]interface{}{
			"Name":                smcpName,
			"Version":             smcpVer.String(),
			"Member":              ns.Foo,
			"manageNetworkPolicy": "false",
			"Rosa":                env.IsRosa(),
		}
		kialiValues := map[string]string{
			"SmcpName":      smcpName,
			"SmcpNamespace": meshNamespace,
			"KialiVersion":  env.GetKialiVersion(),
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
		oc.WaitPodRunning(t, pod.MatchingSelector("app=kiali", meshNamespace))
		kialiPodName := shell.Executef(t, "oc get pod -l app=kiali -n %s -o jsonpath='{.items[0].metadata.name}'", meshNamespace)

		t.LogStep("Grant cluster-monitoring-view to Kiali")
		oc.ApplyTemplate(t, meshNamespace, kialiClusterMonitoringView, kialiValues)

		t.NewSubTest("SMCP manageNetworkPolicy false").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, istiodMonitor)
				oc.DeleteFromString(t, ns.Foo, istioProxyMonitor)
				oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
				app.Uninstall(t, app.Httpbin(ns.Foo))
				oc.DeleteFromTemplate(t, meshNamespace, meshTmpl, meshValues)
				oc.RecreateNamespace(t, ns.Foo)
				oc.RecreateNamespace(t, meshNamespace)
			})

			t.LogStep("Deploying SMCP")
			oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			waitKialiAndVerifyIsReconciled(t)
			t.LogStep("Wait until the old Kiali pod has been deleted")
			oc.WaitUntilResourceExist(t, meshNamespace, "pod", kialiPodName)
			kialiPodName = shell.Executef(t, "oc get pod -l app=kiali -n %s -o jsonpath='{.items[0].metadata.name}'", meshNamespace)

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

			waitUntilAllPrometheusTargetReady(t, kialiToken)
			generateTrafficAndcheckMetrics(t, kialiToken)
		})

		t.NewSubTest("SMCP manageNetworkPolicy true").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, istiodMonitor)
				oc.DeleteFromString(t, ns.Foo, istioProxyMonitor)
				oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
				app.Uninstall(t, app.Httpbin(ns.Foo))
				oc.DeleteFromTemplate(t, meshNamespace, networkPolicy, map[string]string{"namespace": meshNamespace})
				oc.DeleteFromTemplate(t, meshNamespace, meshTmpl, meshValues)
				oc.RecreateNamespace(t, ns.Foo)
				oc.RecreateNamespace(t, meshNamespace)
			})

			t.LogStep("Deploying SMCP")
			meshValues["manageNetworkPolicy"] = "true"
			oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)

			waitKialiAndVerifyIsReconciled(t)
			t.LogStep("Wait until the old Kiali pod has been deleted")
			oc.WaitUntilResourceExist(t, meshNamespace, "pod", kialiPodName)
			kialiPodName = shell.Executef(t, "oc get pod -l app=kiali -n %s -o jsonpath='{.items[0].metadata.name}'", meshNamespace)

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

			waitUntilAllPrometheusTargetReady(t, kialiToken)
			generateTrafficAndcheckMetrics(t, kialiToken)
		})
	})
}

func waitKialiAndVerifyIsReconciled(t TestHelper) {
	t.LogStep("Wait until Kiali is ready")
	oc.WaitCondition(t, meshNamespace, "Kiali", kialiName, "Successful")

	t.LogStep("Verify that Kiali was reconciled by Istio Operator")
	retry.UntilSuccess(t, func(t TestHelper) {
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

func generateTrafficAndcheckMetrics(t TestHelper, thanosToken string) {
	t.LogStep("Generate some ingress traffic")
	oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.6/samples/httpbin/httpbin-gateway.yaml")
	httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))

	for i := 0; i < 5; i++ {
		retry.UntilSuccess(t, func(t TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})
	}

	t.LogStep("Check istiod metrics")
	checkMetricExists(t, meshNamespace, "pilot_info", thanosToken)

	t.LogStep("Check httpbin metrics")
	checkMetricExists(t, ns.Foo, "istio_requests_total", thanosToken)
}

func checkMetricExists(t TestHelper, ns, metricName, token string) {
	retry.UntilSuccess(t, func(t TestHelper) {
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

func waitUntilAllPrometheusTargetReady(t TestHelper, token string) {
	waitUntilPrometheusTargetReady(t, "serviceMonitor", meshNamespace, "istiod-monitor", token)
	waitUntilPrometheusTargetReady(t, "podMonitor", ns.Foo, "istio-proxies-monitor", token)
}

func waitUntilPrometheusTargetReady(t TestHelper, monitorType string, ns string, targetName string, token string) {
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.Exec(t,
			pod.MatchingSelectorFirst("app.kubernetes.io/instance=thanos-querier", monitoringNs),
			"thanos-query",
			prometheusActiveTargetQuery(token),
			assert.OutputContains(
				fmt.Sprintf(`"scrapePool":"%s/%s/%s`, monitorType, ns, targetName),
				fmt.Sprintf("The %s %s prometheus target is ready in namespace %s", monitorType, targetName, ns),
				fmt.Sprintf("The %s %s prometheus target is not ready yet in namespace %s", monitorType, targetName, ns)),
		)
	})
}

func prometheusQuery(ns, metricName, token string) string {
	return fmt.Sprintf(
		`curl -X GET -kG "https://localhost:9091/api/v1/query?namespace=%s&query=%s" --data-urlencode "query=up" -H "Authorization: Bearer %s"`,
		ns, metricName, token)
}

func prometheusActiveTargetQuery(token string) string {
	return fmt.Sprintf(
		`curl -X GET -kG "https://localhost:9091/api/v1/targets?state=active" --data-urlencode "query=up" -H "Authorization: Bearer %s"`, token)
}

func fetchKialiToken(t TestHelper) string {
	var kialiToken string
	retry.UntilSuccess(t, func(t TestHelper) {
		kialiToken = shell.Executef(t, "oc exec -n %s $(oc get pods -n %s -l app=kiali -o jsonpath='{.items[].metadata.name}') "+
			"-- cat /var/run/secrets/kubernetes.io/serviceaccount/token", meshNamespace, meshNamespace)
		if strings.Contains(kialiToken, "Error") {
			t.Errorf("unexpected error: %s", kialiToken)
		}
	})
	return kialiToken
}

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
	kialiName                = "kiali-user-workload-monitoring"
	monitoringNs             = "openshift-monitoring"
	userWorkloadMonitoringNs = "openshift-user-workload-monitoring"
	thanosTokenPrefix        = "prometheus-user-workload-token-"
	thanosTokenSecret        = "thanos-querier-web-token"
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
		meshValues := map[string]string{
			"Name":    smcpName,
			"Version": smcpVer.String(),
			"Member":  ns.Foo,
		}
		kialiValues := map[string]string{
			"SmcpName":      smcpName,
			"SmcpNamespace": meshNamespace,
		}

		t.Cleanup(func() {
			oc.DeleteFromString(t, meshNamespace, istiodMonitor)
			oc.DeleteFromString(t, ns.Foo, istioProxyMonitor)
			oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
			oc.DeleteFromTemplate(t, monitoringNs, clusterMonitoringConfigTmpl, map[string]bool{"Enabled": false})
			oc.DeleteFromString(t, meshNamespace, enableTrafficMetrics)
			app.Uninstall(t, app.Httpbin(ns.Foo))
			oc.DeleteFromTemplate(t, meshNamespace, meshTmpl, meshValues)
			oc.DeleteFromTemplate(t, meshNamespace, kialiUserWorkloadMonitoringTmpl, kialiValues)
			oc.DeleteSecret(t, meshNamespace, thanosTokenSecret)
		})

		t.LogStep("Waiting until user workload monitoring stack is up and running")
		oc.ApplyTemplate(t, monitoringNs, clusterMonitoringConfigTmpl, map[string]bool{"Enabled": true})
		oc.WaitPodsExist(t, userWorkloadMonitoringNs)
		oc.WaitAllPodsReady(t, userWorkloadMonitoringNs)

		t.LogStep("Fetch Thanos secret")
		var secret string
		retry.UntilSuccess(t, func(t test.TestHelper) {
			secret = shell.Executef(t,
				`kubectl get secrets --no-headers -o custom-columns=":metadata.name" -n %s | grep %s`, userWorkloadMonitoringNs, thanosTokenPrefix)
			if strings.Contains(secret, "Error") {
				t.Errorf("unexpected error: %s", secret)
			}
			// secret name contains '\n', because it's parsed from a single output column
			secret = strings.TrimSuffix(secret, "\n")
		})

		t.LogStep("Fetch Thanos token")
		var thanosToken string
		retry.UntilSuccess(t, func(t test.TestHelper) {
			thanosToken = shell.Executef(t, "oc get secret %s -n %s --template={{.data.token}} | base64 -d", secret, userWorkloadMonitoringNs)
			if strings.Contains(thanosToken, "Error") {
				t.Errorf("unexpected error: %s", thanosToken)
			}
		})

		t.LogStep("Create secret with Thanos token for Kiali")
		shell.Executef(t, "oc create secret generic %s -n %s --from-literal=token=%s", thanosTokenSecret, meshNamespace, thanosToken)

		t.LogStep("Deploying Kiali")
		oc.ApplyTemplate(t, meshNamespace, kialiUserWorkloadMonitoringTmpl, kialiValues)

		t.LogStep("Deploying SMCP")
		oc.ApplyTemplate(t, meshNamespace, meshTmpl, meshValues)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Wait until Kiali is ready")
		oc.WaitKialiSuccessful(t, meshNamespace, kialiName)

		t.LogStep("Verify that Kiali was reconciled by Istio Operator")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			output := shell.Executef(t, "oc get kiali %s -n %s -o jsonpath='{.spec.deployment.accessible_namespaces}'", kialiName, meshNamespace)
			if output != fmt.Sprintf(`["%s"]`, ns.Foo) {
				t.Errorf(`unexpected accessible namespaces: got '%s', expected: '["%s"]'`, output, ns.Foo)
			}
		})

		t.LogStep("Enable Prometheus telemetry")
		oc.ApplyString(t, meshNamespace, enableTrafficMetrics)

		t.LogStep("Deploy httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))
		oc.WaitPodsExist(t, ns.Foo)
		oc.WaitAllPodsReady(t, ns.Foo)

		t.LogStep("Apply Prometheus monitors")
		oc.ApplyString(t, meshNamespace, istiodMonitor)
		oc.ApplyString(t, ns.Foo, istioProxyMonitor)

		t.LogStep("Generate some ingress traffic")
		oc.ApplyFile(t, ns.Foo, "https://raw.githubusercontent.com/maistra/istio/maistra-2.4/samples/httpbin/httpbin-gateway.yaml")
		httpbinURL := fmt.Sprintf("http://%s/headers", istio.GetIngressGatewayHost(t, meshNamespace))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			curl.Request(t, httpbinURL, nil, assert.ResponseStatus(http.StatusOK))
		})

		t.LogStep("Check istiod metrics")
		checkMetricExists(t, meshNamespace, "pilot_info", thanosToken)

		t.LogStep("Check httpbin metrics")
		checkMetricExists(t, ns.Foo, "istio_requests_total", thanosToken)
	})
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
		`curl -X GET -kG "https://localhost:9092/api/v1/query?namespace=%s&query=%s" --data-urlencode "query=up" -H "Authorization: Bearer %s"`,
		ns, metricName, token)
}

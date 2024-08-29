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

package istio

import (
	"strings"

	prometheus "github.com/prometheus/client_model/go"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"

	"github.com/prometheus/common/expfmt"
)

func GetIngressGatewayHost(t test.TestHelper, meshNamespace string) string {
	return shell.Executef(t, "kubectl -n %s get routes istio-ingressgateway -o jsonpath='{.spec.host}'", meshNamespace)
}

func GetIngressGatewaySecurePort(t test.TestHelper, meshNamespace string) string {
	return shell.Executef(t, `kubectl -n %s get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="https")].port}'`, meshNamespace)
}

func GetProxyMetric(t test.TestHelper, oc *oc.OC, podLocator oc.PodLocatorFunc, metric string, labels ...string) *prometheus.Metric {
	t.T().Helper()
	metrics := GetProxyMetrics(t, oc, podLocator, metric, labels...)
	switch len(metrics) {
	case 0:
		return nil
	case 1:
		return metrics[0]
	default:
		t.Fatalf("multiple metrics named %q matched the given labels: %v", metric, labels)
		return nil
	}
}

func GetProxyMetrics(t test.TestHelper, oc *oc.OC, podLocator oc.PodLocatorFunc, metric string, labels ...string) []*prometheus.Metric {
	t.T().Helper()
	output := oc.Exec(t, podLocator, "istio-proxy", "curl -sS localhost:15000/stats/prometheus")

	var parser expfmt.TextParser
	families, err := parser.TextToMetricFamilies(strings.NewReader(output))
	if err != nil {
		t.Fatalf("could not parse Prometheus metrics: %v", err)
	}

	family, exists := families[metric]
	if !exists {
		return nil
	}

	var matchedMetrics []*prometheus.Metric
	for _, m := range family.Metric {
		if matchesLabels(m, labels) {
			matchedMetrics = append(matchedMetrics, m)
		}
	}
	return matchedMetrics
}

func matchesLabels(metric *prometheus.Metric, labels []string) bool {
	metricLabels := map[string]string{}
	for _, l := range metric.Label {
		metricLabels[*l.Name] = *l.Value
	}

	for _, l := range labels {
		arr := strings.SplitN(l, "=", 2)
		if metricLabels[arr[0]] != arr[1] {
			return false
		}
	}
	return true
}

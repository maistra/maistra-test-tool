package prometheus

import (
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var DefaultPrometheus = NewPrometheus("app=prometheus", "prometheus")

var DefaultCustomPrometheus = DefaultPrometheus.
	WithSelector("prometheus=prometheus")

var DefaultThanos = DefaultPrometheus.
	WithSelector("app.kubernetes.io/instance=thanos-querier").
	WithContainerName("thanos-query")

type PrometheusResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type PrometheusResultData struct {
	ResultType string             `json:"resultType"`
	Result     []PrometheusResult `json:"result"`
}

type PrometheusResponse struct {
	Status string               `json:"status"`
	Data   PrometheusResultData `json:"data"`
}

type Prometheus interface {
	WithSelector(selector string) Prometheus
	WithContainerName(containerName string) Prometheus
	Query(t test.TestHelper, ns string, query string) PrometheusResponse
}

func Query(t test.TestHelper, ns string, query string) PrometheusResponse {
	return DefaultPrometheus.Query(t, ns, query)
}

func CustomPrometheusQuery(t test.TestHelper, ns string, query string) PrometheusResponse {
	return DefaultCustomPrometheus.Query(t, ns, query)
}

func ThanosQuery(t test.TestHelper, ns string, query string) PrometheusResponse {
	return DefaultThanos.Query(t, ns, query)
}

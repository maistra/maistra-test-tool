package prometheus

import (
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var DefaultPrometheus = NewPrometheus("app=prometheus")
var CustomPrometheus = NewPrometheus("prometheus=prometheus")

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
	Query(t test.TestHelper, ns string, query string) PrometheusResponse
}

func Query(t test.TestHelper, ns string, query string) PrometheusResponse {
	return DefaultPrometheus.Query(t, ns, query)
}

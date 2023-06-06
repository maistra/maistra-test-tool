package prometheus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type prometheus_struct struct {
	selector string
}

var _ Prometheus = &prometheus_struct{}

func NewPrometheus(selector string) Prometheus {
	return &prometheus_struct{selector}
}

func (pi *prometheus_struct) Query(t test.TestHelper, ns string, query string) PrometheusResponse {
	escapedQuery := strings.ReplaceAll(query, `'`, `'\\''`)

	output := oc.Exec(t,
		pod.MatchingSelector(pi.selector, ns), "prometheus",
		fmt.Sprintf("curl -sS localhost:9090/api/v1/query --data-urlencode 'query=%s'", escapedQuery))

	result := &PrometheusResponse{}
	err := json.Unmarshal([]byte(output), result)
	if err != nil {
		t.Log("Prometheus response:\n%s", output)
		t.Fatalf("could not parse Prometheus response as JSON: %v", err)
	}

	return *result
}

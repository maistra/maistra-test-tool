package prometheus

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

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

func Query(t test.TestHelper, ns string, query string) PrometheusResponse {
	escapedQuery := strings.ReplaceAll(query, `'`, `'\\''`)

	output := oc.Exec(t,
		pod.MatchingSelector("app=prometheus", ns), "prometheus",
		fmt.Sprintf("curl -sS localhost:9090/api/v1/query --data-urlencode 'query=%s'", escapedQuery))

	result := &PrometheusResponse{}
	err := json.Unmarshal([]byte(output), result)
	if err != nil {
		t.Log("Prometheus response:\n%s", output)
		t.Fatalf("could not parse Prometheus response as JSON: %v", err)
	}

	return *result
}

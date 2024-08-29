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

package prometheus

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func NewPrometheus(selector, containerName string) Prometheus {
	return &prometheus_struct{selector, containerName}
}

type prometheus_struct struct {
	selector      string
	containerName string
}

func (pi *prometheus_struct) clone() *prometheus_struct {
	new := *pi
	return &new
}

var _ Prometheus = &prometheus_struct{}

func (pi *prometheus_struct) WithSelector(selector string) Prometheus {
	new := pi.clone()
	new.selector = selector
	return new
}

func (pi *prometheus_struct) WithContainerName(containerName string) Prometheus {
	new := pi.clone()
	new.containerName = containerName
	return new
}

func (pi *prometheus_struct) Query(t test.TestHelper, ns string, query string) PrometheusResponse {
	queryString := url.Values{"query": []string{query}}.Encode()
	url := fmt.Sprintf(`http://localhost:9090/api/v1/query?%s`, queryString)
	urlShellEscaped := strings.ReplaceAll(url, `'`, `'\\''`)

	output := oc.Exec(t,
		pod.MatchingSelectorFirst(pi.selector, ns), pi.containerName,
		fmt.Sprintf("curl -sS -X GET '%s'", urlShellEscaped))

	return parsePrometheusResponse(t, output)
}

func parsePrometheusResponse(t test.TestHelper, response string) PrometheusResponse {
	result := &PrometheusResponse{}
	err := json.Unmarshal([]byte(response), result)
	if err != nil {
		t.Log("Prometheus response:\n%s", response)
		t.Fatalf("could not parse Prometheus response as JSON: %v", err)
	}

	return *result
}

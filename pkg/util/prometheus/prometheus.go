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

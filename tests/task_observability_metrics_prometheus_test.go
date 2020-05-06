// Copyright 2020 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"fmt"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupPrometheusMetrics(namespace string) {
	log.Info("# Cleanup ...")
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestPrometheusMetrics(t *testing.T) {
	defer cleanupPrometheusMetrics(testNamespace)
	defer recoverPanic(t)

	log.Info("Prometheus Collecting Metrics")
	deployBookinfo(testNamespace, false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	prometheusRoute, _ := util.Shell("kubectl get routes -n %s -l app=prometheus -o jsonpath='{.items[0].spec.host}'", meshNamespace)

	t.Run("Observability_metrics_prometheus_collecting_metrics", func(t *testing.T) {
		defer recoverPanic(t)

		if err := util.KubeApply(meshNamespace, bookinfoMetrics, kubeconfig); err != nil {
			t.Errorf("Failed to apply bookinfo metrics")
			log.Errorf("Failed to apply bookinfo metrics")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		for i := 0; i <= 5; i++ {
			util.GetHTTPResponse(productpageURL, nil)
		}

		// TBD Oauth and UI automation
		log.Infof("Access the Prometheus dashboard: %s", prometheusRoute)
		log.Info("Query Execute 'istio_double_request_count'")

		query := "istio_double_request_count"
		queryURL := fmt.Sprintf("https://%s/graph?g0.range_input=1h&g0.expr=%s&g0.tab=1", prometheusRoute, query)
		resp, _, err := util.GetHTTPResponse(queryURL, nil)
		log.Infof("Got response: %s", err)
		util.CloseResponseBody(resp)
		time.Sleep(time.Duration(waitTime*6) * time.Second)
	})
}

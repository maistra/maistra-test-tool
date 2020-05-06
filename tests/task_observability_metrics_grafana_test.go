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

func cleanupGrafanaMetrics(namespace string) {
	log.Info("# Cleanup ...")
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestGrafanaMetrics(t *testing.T) {
	defer cleanupGrafanaMetrics(testNamespace)
	defer recoverPanic(t)

	log.Info("Visualizing Metrics with Grafana")
	deployBookinfo(testNamespace, false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	grafanaRoute, _ := util.Shell("kubectl get routes -n %s -l app=grafana -o jsonpath='{.items[0].spec.host}'", meshNamespace)

	t.Run("Observability_metrics_grafana_visualizing_metrics", func(t *testing.T) {
		defer recoverPanic(t)

		for i := 0; i <= 5; i++ {
			util.GetHTTPResponse(productpageURL, nil)
		}

		// TBD Oauth and UI automation
		log.Infof("Access the Grafana dashboard: %s", grafanaRoute)
		log.Info("Check istio-mesh-dashboard, istio-service-dashboard and istio-workload-dashboard")

		// https://grafana-istio-system.apps.yuaxu-maistra-4.4.devcluster.openshift.com/d/G8wLrJIZk/istio-mesh-dashboard
		// https://grafana-istio-system.apps.yuaxu-maistra-4.4.devcluster.openshift.com/d/LJ_uJAvmk/istio-service-dashboard?var-service=productpage.bookinfo.svc.cluster.local
		// https://grafana-istio-system.apps.yuaxu-maistra-4.4.devcluster.openshift.com/d/UbsSZTDik/istio-workload-dashboard?var-namespace=bookinfo

		metricURL := fmt.Sprintf("https://%s/", grafanaRoute)
		resp, _, err := util.GetHTTPResponse(metricURL, nil)
		log.Infof("Got response: %s", err)
		util.CloseResponseBody(resp)
		time.Sleep(time.Duration(waitTime*6) * time.Second)
	})
}

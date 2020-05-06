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

func cleanupJaegerTracing(namespace string) {
	log.Info("# Cleanup ...")
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestJaegerTracing(t *testing.T) {
	defer cleanupJaegerTracing(testNamespace)
	defer recoverPanic(t)

	log.Info("Distributed Tracing Jaeger")
	deployBookinfo(testNamespace, false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)
	jaegerRoute, _ := util.Shell("kubectl get routes -n %s -l app=jaeger -o jsonpath='{.items[0].spec.host}'", meshNamespace)

	t.Run("Observability_check_jaeger_dashboard", func(t *testing.T) {
		defer recoverPanic(t)

		for i := 0; i <= 5; i++ {
			util.GetHTTPResponse(productpageURL, nil)
		}

		// TBD Oauth and UI automation
		log.Infof("Access the Jaeger dashboard: %s", jaegerRoute)
		log.Info("Select Service 'productpage' and 'Find Traces'")

		searchURL := fmt.Sprintf("https://%s/search?service=productpage.%s", jaegerRoute, testNamespace)
		resp, _, err := util.GetHTTPResponse(searchURL, nil)
		log.Infof("Got response: %s", err)
		util.CloseResponseBody(resp)
		time.Sleep(time.Duration(waitTime*6) * time.Second)
	})
}

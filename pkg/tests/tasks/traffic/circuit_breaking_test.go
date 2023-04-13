// Copyright 2021 Red Hat, Inc.
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

package traffic

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestCircuitBreaking(t *testing.T) {
	NewTest(t).Id("T6").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Fortio(ns))

		t.Log("verify traffic management tripping circuit breaker")
		t.LogStep("Configure circuit breaker destination rule")
		oc.ApplyString(t, ns, httpbinCircuitBreaker)

		t.LogStep("Verify connection with curl: expected 200 OK")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=fortio", ns),
				"fortio",
				"/usr/bin/fortio curl -quiet http://httpbin:8000/get",
				assert.OutputContains("200",
					"Got expected response from httpbin: 200 OK",
					"ERROR: Got unexpected response from httpbin not 200 OK"))
		})

		t.LogStep("Tripping the circuit breaker")
		t.Log("To tipping the circuit breaker we are going to send 50 requests to httpbin with 2 connections")
		connection := 2
		reqCount := 50
		msg := oc.Exec(t,
			pod.MatchingSelector("app=fortio", ns),
			"fortio",
			fmt.Sprintf("/usr/bin/fortio load -c %d -qps 0 -n %d -loglevel Warning http://httpbin:8000/get", connection, reqCount))

		t.LogStep("Validate the number of 200 responses")
		t.Log("verify from output message the number of 200 responses to the load test. We expect to have a line with this information: Code 200 : XX (X0.0 %)")
		c200 := getNumberOfResponses(t, msg, `Code 200.*`)

		t.LogStep("Validate the number of 503 responses")
		t.Log("verify from output message the number of 500 responses to the load test. We expect to have a line with this information: Code 503 : XX (X0.0 %)")
		c503 := getNumberOfResponses(t, msg, `Code 503.*`)

		t.LogStep("Validate the percentage of 200 responses and 503 to the total of requests")
		t.Log("We expect to have 60% 200 responses and 40% 503 responses")
		tolerance := 0.1
		successRate200 := float64(c200) / float64(reqCount) * 100
		successRate503 := float64(c503) / float64(reqCount) * 100
		retry.UntilSuccess(t, func(t test.TestHelper) {
			if util.IsWithinPercentage(c200, reqCount, 0.6, tolerance) && util.IsWithinPercentage(c503, reqCount, 0.4, tolerance) {
				t.Logf(
					"Success. Circuit breaking acts as expected. "+
						"Code 200 hit %d of %d (%.2f%%), Code 503 hit %d of %d (%.2f%%)",
					c200, reqCount, successRate200, c503, reqCount, successRate503)
			} else {
				t.Fatalf(
					"Failed Circuit breaking. "+
						"Code 200 hit %d 0f %d, Code 503 hit %d of %d", c200, reqCount, c503, reqCount)
			}
		})
	})
}

func getNumberOfResponses(t test.TestHelper, msg string, codeText string) int {
	re := regexp.MustCompile(codeText)
	line := re.FindStringSubmatch(msg)[0]
	re = regexp.MustCompile(`: [\d]+`)
	word := re.FindStringSubmatch(line)[0]
	count, err := strconv.Atoi(strings.TrimLeft(word, ": "))
	if err != nil {
		t.Fatalf("Failed to parse %s count: %v", codeText, err)
	}

	return count
}

var httpbinCircuitBreaker = `
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: httpbin
spec:
  host: httpbin
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 1
      http:
        http1MaxPendingRequests: 1
        maxRequestsPerConnection: 1
    outlierDetection:
      consecutiveErrors: 1
      interval: 1s
      baseEjectionTime: 3m
      maxEjectionPercent: 100
`

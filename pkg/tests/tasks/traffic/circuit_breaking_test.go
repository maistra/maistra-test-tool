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
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestCircuitBreaking(t *testing.T) {
	NewTest(t).Id("T6").Groups(Full, InterOp).Run(func(t TestHelper) {
		t.Log("This test checks whether the circuit breaker functions correctly")

		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and fortio")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Fortio(ns))

		t.LogStep("Configure circuit breaker destination rule")
		oc.ApplyString(t, ns, httpbinCircuitBreaker)

		t.LogStep("Verify connection with curl: expect 200 OK")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=fortio", ns),
				"fortio",
				"/usr/bin/fortio curl -quiet http://httpbin:8000/get",
				assert.OutputContains("200",
					"Got expected 200 OK response from httpbin",
					"Expected 200 OK from httpbin, but got an unexpected response"))
		})

		connection := 2
		reqCount := 50
		tolerance := 0.1
		t.LogStep("Trip the circuit breaker by sending 50 requests to httpbin with 2 connections")
		t.Logf("We expect 60%% of responses to return 200 OK, and 40%% to return 503 Service Unavailable (tolerance %d%%)", int(100*tolerance))
		retry.UntilSuccess(t, func(t test.TestHelper) {
			msg := oc.Exec(t,
				pod.MatchingSelector("app=fortio", ns),
				"fortio",
				fmt.Sprintf("/usr/bin/fortio load -c %d -qps 0.1 -n %d -loglevel Warning http://httpbin:8000/get", connection, reqCount))

			c200 := getNumberOfResponses(t, msg, `Code 200.*`)
			c503 := getNumberOfResponses(t, msg, `Code 503.*`)
			successRate200 := 100 * c200 / reqCount
			successRate503 := 100 * c503 / reqCount

			if util.IsWithinPercentage(c200, reqCount, 0.6, tolerance) && util.IsWithinPercentage(c503, reqCount, 0.4, tolerance) {
				t.LogSuccessf("%d%% of responses were 200 OK, and %d%% were 503 Service Unavailable", successRate200, successRate503)
			} else {
				t.Fatalf("%d%% of responses were 200 OK, and %d%% were 503 Service Unavailable", successRate200, successRate503)
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

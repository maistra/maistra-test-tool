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
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestCircuitBreaking(t *testing.T) {
	NewTest(t).Id("T6").Groups(Full, InterOp).Run(func(t TestHelper) {
		t.Log("This test checks whether the circuit breaker functions correctly. Check documentation: https://istio.io/latest/docs/tasks/traffic-management/circuit-breaking/")

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
		t.LogStep("Trip the circuit breaker by sending 50 requests to httpbin with 2 connections")
		t.Log("We expect request with response code 503")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			msg := oc.Exec(t,
				pod.MatchingSelector("app=fortio", ns),
				"fortio",
				fmt.Sprintf("/usr/bin/fortio load -c %d -qps 0 -n %d -loglevel Warning http://httpbin:8000/get", connection, reqCount))

			c200 := getNumberOfResponses(t, msg, `Code 200.*`)
			c503 := getNumberOfResponses(t, msg, `Code 503.*`)
			successRate200 := 100 * c200 / reqCount
			successRate503 := 100 * c503 / reqCount
			t.Log(fmt.Sprintf("Success rate 200: %d%%", successRate200))
			t.Log(fmt.Sprintf("Success rate 503: %d%%", successRate503))

			t.LogStep("Validate the circuit breaker is tripped by checking the istio-proxy log")
			t.Log("Verify istio-proxy pilot-agent stats, expected upstream_rq_pending_overflow value to be more than zero")
			output := oc.Exec(t,
				pod.MatchingSelector("app=fortio", ns),
				"istio-proxy",
				"pilot-agent request GET stats | grep httpbin | grep pending")
			assertProxyContainsUpstreamRqPendingOverflow(t, output)
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

func assertProxyContainsUpstreamRqPendingOverflow(t test.TestHelper, output string) {
	var v int
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "upstream_rq_pending_overflow") {
			parts := strings.Split(line, ": ")
			var err error
			v, err = strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				t.Errorf("failed to parse upstream_rq_pending_overflow value: %v", err)
			}
			if v > 0 {
				t.LogSuccessf("Found Upstream_rq_pending_overflow : %d", v)
				break
			}
		}
	}
	if v == 0 {
		t.Errorf("failed to get upstream_rq_pending_overflow value: %v", v)
	}
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
`

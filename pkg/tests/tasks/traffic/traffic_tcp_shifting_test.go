// Copyright 2023 Red Hat, Inc.
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
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestTcpTrafficShifting validates TCP traffic shifting feature.
func TestTcpTrafficShifting(t *testing.T) {
	test.NewTest(t).Id("T4").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		ns := "foo"

		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns), app.EchoV1(ns), app.EchoV2(ns))
		})

		t.Log("This test validates traffic shifting for TCP traffic.")
		t.Log("Doc reference: https://istio.io/v1.14/docs/tasks/traffic-management/tcp-traffic-shifting/")

		t.LogStep("Install sleep, echoV1 and echoV2")
		app.InstallAndWaitReady(t, app.Sleep(ns), app.EchoV1(ns), app.EchoV2(ns))

		t.NewSubTest("tcp shift 100 percent to v1").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, EchoAllv1Yaml)
			})

			t.LogStep("Shifting all TCP traffic to v1")
			oc.ApplyString(t, ns, EchoAllv1Yaml)

			t.LogStep("make 20 requests and checking if all of them go to v1 (tolerance: 0%)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				tolerance := 0.0
				checkTcpTrafficRatio(t, ns, "tcp-echo", "9000", 20, tolerance, map[string]float64{
					"one": 1.0,
					"two": 0.0,
				})
			})
		})

		t.NewSubTest("tcp shift 20 percent to v2").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, Echo20v2Yaml)
			})

			t.LogStep("Shifting 20 percent TCP traffic to v2")
			oc.ApplyString(t, ns, Echo20v2Yaml)

			t.LogStep("make 100 requests and checking if 20 percent of them go to v2 (tolerance: 10%)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				tolerance := 0.10
				checkTcpTrafficRatio(t, ns, "tcp-echo", "9000", 100, tolerance, map[string]float64{
					"one": 0.8,
					"two": 0.2,
				})
			})
		})
	})
}

func checkTcpTrafficRatio(t test.TestHelper, ns, host, port string, numberOfRequests int, tolerance float64, ratios map[string]float64) {
	counts := map[string]int{}
	oc.Exec(t,
		pod.MatchingSelector("app=sleep", ns),
		"sleep",
		fmt.Sprintf(`sh -c 'i=1; while [ $i -le %d ]; do date | nc %s %s; i=$((i+1)); done'`, numberOfRequests, host, port),
		func(t test.TestHelper, output string) {
			scanner := bufio.NewScanner(strings.NewReader(output))
			for scanner.Scan() {
				line := scanner.Text()
				matched := false
				for version := range ratios {
					if strings.Contains(line, version) {
						matched = true
						counts[version]++
					}
				}
				if !matched {
					t.Fatalf("nc tcp-echo output did not match any expected version, got output: %s", output)
				}
			}
		})

	for version, count := range counts {
		expectedRate := ratios[version]
		actualRate := float64(count) / float64(numberOfRequests)
		if util.IsWithinPercentage(count, numberOfRequests, expectedRate, tolerance) {
			t.Logf("success: %d/%d responses matched %s (actual rate %f, expected %f, tolerance %f)", count, numberOfRequests, version, actualRate, expectedRate, tolerance)
		} else {
			t.Errorf("failure: %d/%d responses matched %s (actual rate %f, expected %f, tolerance %f)", count, numberOfRequests, version, actualRate, expectedRate, tolerance)
		}
	}
}

const (
	EchoAllv1Yaml = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: tcp-echo-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 31400
      name: tcp
      protocol: TCP
    hosts:
    - "*"
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: tcp-echo-destination
spec:
  host: tcp-echo
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: tcp-echo
spec:
  hosts:
  - tcp-echo
  tcp:
  - route:
    - destination:
        host: tcp-echo
        port:
          number: 9000
        subset: v1	
`
	Echo20v2Yaml = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: tcp-echo-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 31400
      name: tcp
      protocol: TCP
    hosts:
    - "*"
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: tcp-echo-destination
spec:
  host: tcp-echo
  subsets:
  - name: v1
    labels:
      version: v1
  - name: v2
    labels:
      version: v2
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: tcp-echo
spec:
  hosts:
  - tcp-echo
  tcp:
  - route:
    - destination:
        host: tcp-echo
        port:
          number: 9000
        subset: v1
      weight: 80
    - destination:
        host: tcp-echo
        port:
          number: 9000
        subset: v2
      weight: 20	
`
)

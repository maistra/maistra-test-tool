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

package egress

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAccessExternalServices(t *testing.T) {
	test.NewTest(t).Id("T11").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "bookinfo"
		meshNamespace := env.GetDefaultMeshNamespace()
		smcpName := env.GetDefaultSMCPName()
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
			oc.Patch(t,
				meshNamespace,
				"smcp", smcpName,
				"json", `[{"op": "remove", "path": "/spec/proxy"}]`)
		})

		t.Log("This test validates accesses to external services")

		assertRequestSuccess := func(url string) {
			execInSleepPod(t, ns, buildGetRequestCmd(url),
				assert.OutputContains("200",
					fmt.Sprintf("Got expected 200 OK from %s", url),
					fmt.Sprintf("Expect 200 OK from %s, but got a different HTTP code", url)))
		}
		assertRequestFailure := func(url string) {
			execInSleepPod(t, ns, buildGetRequestCmd(url),
				assert.OutputContains(curlFailedMessage,
					"Got a failure message as expected",
					"Expect request to failed, but got a response"))
		}
		assertRequestTimeout := func(url string) {
			execInSleepPod(t, ns, buildGetRequestCmd(url),
				assert.OutputContains("504",
					"Got expected 504 response since the request was timeout",
					"Expect a timeout response with 504, but got a different one"))
		}

		t.LogStepf("Install sleep into %s", ns)
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.LogStep("Make request to www.redhat.com from sleep")
		assertRequestSuccess("https://www.redhat.com/en")

		t.LogStepf("Patch outbound traffic policy to registry only")
		oc.Patch(t,
			meshNamespace,
			"smcp", smcpName,
			"json", `
- op: add
  path: /spec/proxy
  value:
    networking:
      trafficControl:
        outbound:
          policy: "REGISTRY_ONLY"`,
		)

		t.LogStep("Make request to www.redhat.com from sleep again, and expect it denied")
		assertRequestFailure("https://www.redhat.com/en")

		t.NewSubTest("allow request to www.redhat.com after applying ServiceEntry").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, redhatExternalServiceEntryHttpsPortOnly)
			})

			t.LogStep("Apply a ServiceEntry to redhat.com")
			oc.ApplyString(t, ns, redhatExternalServiceEntryHttpsPortOnly)

			t.LogStep("Send a request to redhat.com on HTTPS port")
			assertRequestSuccess("https://www.redhat.com/en")
		})

		t.NewSubTest("follow access policies for httpbin.org").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, httpbinExternalServiceEntryHttpPortOnly)
				oc.DeleteFromString(t, ns, httpbinExternalVituralServiceWithTimeout)
			})

			t.LogStep("Apply a ServiceEntry to httpbin.org with only HTTP port")
			oc.ApplyString(t, ns, httpbinExternalServiceEntryHttpPortOnly)

			t.LogStep("Send a request to httpbin.org on HTTP port")
			assertRequestSuccess("http://httpbin.org/headers")

			t.LogStep("Send a request to httpbin.org on HTTPS port")
			assertRequestFailure("https://httpbin.org/headers")

			t.LogStep("Apply a VirtualService with 3-second timeout to httpbin.org")
			oc.ApplyString(t, ns, httpbinExternalVituralServiceWithTimeout)

			t.LogStep("Send a request to httpbin.org with 5-second expected delay")
			assertRequestTimeout("http://httpbin.org/delay/5")
		})
	})
}

func buildGetRequestCmd(location string) string {
	return fmt.Sprintf(`curl -sSL -o /dev/null -w "%%{http_code}" %s 2>/dev/null || echo %s`, location, curlFailedMessage)
}

const (
	curlFailedMessage = "CURL_FAILED"

	httpbinExternalServiceEntryHttpPortOnly = `
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: httpbin-ext
spec:
  hosts:
  - httpbin.org
  ports:
  - number: 80
    name: http
    protocol: HTTP
  resolution: DNS
  location: MESH_EXTERNAL
`

	redhatExternalServiceEntryHttpsPortOnly = `
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: redhat
spec:
  hosts:
  - www.redhat.com
  ports:
  - number: 443
    name: https
    protocol: HTTPS
  resolution: DNS
  location: MESH_EXTERNAL
`

	httpbinExternalVituralServiceWithTimeout = `
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: httpbin-ext
spec:
  hosts:
    - httpbin.org
  http:
  - timeout: 3s
    route:
      - destination:
          host: httpbin.org
        weight: 100
`
)

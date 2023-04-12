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
		outboundDefaultPatch := `[{"op": "remove", "path": "/spec/proxy/networking/trafficControl/outbound/policy"}]`
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
			oc.Patch(
				t,
				meshNamespace,
				"smcp",
				smcpName,
				"json",
				outboundDefaultPatch,
			)
		})

		t.Log("This test validates accesses to external services")

		t.LogStepf("Install sleep into %s", ns)
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.LogStep("Make request to www.redhat.com from sleep")
		execInSleepPod(
			t,
			ns,
			buildGetRequestCmd("https://www.redhat.com/en"),
			assert.OutputContains(
				"200",
				"Got expected 200 ok from www.redhat.com",
				"Expect 200 ok from www.redhat.com, but got a different HTTP code",
			),
		)

		outboundRegistryOnlyPatch := `[{"op": "add", "path": "/spec/proxy/networking/trafficControl/outbound/policy", "value": "REGISTRY_ONLY"}]`

		t.LogStepf("Patch outbound traffic policy to registry only")
		oc.Patch(
			t,
			meshNamespace,
			"smcp",
			smcpName,
			"json",
			outboundRegistryOnlyPatch,
		)

		t.LogStep("Make request to www.redhat.com from sleep again, and expect it denied")
		execInSleepPod(
			t,
			ns,
			buildGetRequestCmd("https://www.redhat.com/en"),
			assert.OutputDoesNotContain(
				"200",
				"Got expected non 200 response from www.redhat.com",
				"Expect non 200 from www.redhat.com, but got a different HTTP code",
			),
		)

		t.NewSubTest("allow request to www.redhat.com after applying ServiceEntry").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, redhatExternalServiceEntryHttpsPortOnly)
			})

			t.LogStep("Apply a ServiceEntry to redhat.com")
			oc.ApplyString(t, ns, redhatExternalServiceEntryHttpsPortOnly)

			t.LogStep("Send a request to redhat.com on HTTPS port")
			execInSleepPod(
				t,
				ns,
				buildGetRequestCmd("https://www.redhat.com/en"),
				assert.OutputContains(
					"200",
					"Got expetcted 200 ok from www.redhat.com",
					"Expect 200 ok from www.redhat.com, but got a different HTTP code",
				),
			)
		})

		t.NewSubTest("follow access policies for httpbin.org").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, httpbinExternalServiceEntryHttpPortOnly)
				oc.DeleteFromString(t, ns, httpbinExternalVituralServiceWithTimeout)
			})

			t.LogStep("Apply a ServiceEntry to httpbin.org with only HTTP port")
			oc.ApplyString(t, ns, httpbinExternalServiceEntryHttpPortOnly)

			t.LogStep("Send a request to httpbin.org on HTTP port")
			execInSleepPod(
				t,
				ns,
				buildGetRequestCmd("http://httpbin.org/headers"),
				assert.OutputContains(
					"200",
					"Got expetcted 200 ok from httpbin.org",
					"Expect 200 ok from httpbin.org, but got a different HTTP code",
				),
			)

			t.LogStep("Send a request to httpbin.org on HTTPS port")
			execInSleepPod(
				t,
				ns,
				buildGetRequestCmd("https://httpbin.org/headers"),
				assert.OutputDoesNotContain(
					"200",
					"Got expetcted non 200 response from httpbin.org",
					"Expect non 200 from httpbin.org, but got a different HTTP code",
				),
			)

			t.LogStep("Apply a VirtualService with 3-second timetout to httpbin.org")
			oc.ApplyString(t, ns, httpbinExternalVituralServiceWithTimeout)

			t.LogStep("Send a request to httpbin.org with 5-second expected delay")
			execInSleepPod(
				t,
				ns,
				buildGetRequestCmd("http://httpbin.org/delay/5"),
				assert.OutputContains(
					"504",
					"Got expected 504 response since the request was timeout",
					"Expect a timeout response with 504, but got a different one",
				),
			)
		})
	})
}

func buildGetRequestCmd(location string) string {
	return fmt.Sprintf("curl -sSL -o /dev/null -D - %s | head -n 1", location)
}

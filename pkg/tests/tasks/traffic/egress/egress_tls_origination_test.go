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
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressTLSOrigination(t *testing.T) {
	test.NewTest(t).Id("T12").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		ns := "bookinfo"
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
		})

		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("TrafficManagement_egress_configure_access_to_external_service").Run(func(t test.TestHelper) {
			t.Log("Create a ServiceEntry to external istio.io")
			oc.ApplyString(t, ns, ExServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ExServiceEntry)
			})

			assertRequestSuccess := func(url string) {
				execInSleepPod(t, ns,
					`curl -sSL -o /dev/null -w "%{http_code}" `+url,
					assert.OutputContains("200",
						fmt.Sprintf("Got expected 200 OK from %s", url),
						fmt.Sprintf("Expect 200 OK from %s, but got a different HTTP code", url)))
			}

			assertRequestSuccess("http://istio.io")
		})

		t.NewSubTest("TrafficManagement_egress_tls_origination").Run(func(t test.TestHelper) {
			t.Log("TLS origination for egress traffic")
			oc.ApplyString(t, ns, ExServiceEntryOriginate)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ExServiceEntryOriginate)
			})

			assertRequestSuccess := func(url string) {
				execInSleepPod(t, ns,
					fmt.Sprintf(`curl -sSL -o /dev/null %s -w "%%{http_code}" %s`, getCurlProxyParams(t), url),
					assert.OutputContains("200",
						fmt.Sprintf("Got expected 200 OK from %s", url),
						fmt.Sprintf("Expect 200 OK from %s, but got a different HTTP code", url)))
			}

			assertRequestSuccess("http://istio.io")
			assertRequestSuccess("https://istio.io")
		})
	})
}

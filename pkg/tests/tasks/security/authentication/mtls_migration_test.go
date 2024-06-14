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

package authentication

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMTlsMigration(t *testing.T) {
	test.NewTest(t).Id("T19").Groups(test.Full, test.InterOp, test.ARM).Run(func(t test.TestHelper) {
		meshNamespace := env.GetDefaultMeshNamespace()

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Foo, ns.Bar, ns.Legacy) // TODO: recreate all three namespaces with a single call to RecreateNamespace
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and sleep in multiple namespaces")
		app.InstallAndWaitReady(t,
			app.Httpbin(ns.Foo),
			app.Httpbin(ns.Bar),
			app.Sleep(ns.Foo),
			app.Sleep(ns.Bar),
			app.SleepNoSidecar(ns.Legacy))

		fromNamespaces := []string{ns.Foo, ns.Bar, ns.Legacy}
		toNamespaces := []string{ns.Foo, ns.Bar}

		t.LogStep("Check connectivity from namespaces foo, bar, and legacy to namespace foo and bar")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			for _, from := range fromNamespaces {
				for _, to := range toNamespaces {
					app.AssertSleepPodRequestSuccess(t, from, fmt.Sprintf("http://httpbin.%s:8000/ip", to))
				}
			}
		})

		t.NewSubTest("mTLS enabled in foo").Run(func(t test.TestHelper) {
			t.LogStep("Apply strict mTLS in namespace foo")
			oc.ApplyString(t, "foo", PeerAuthenticationMTLSStrict)

			t.LogStep("Check connectivity from namespaces foo, bar, and legacy to namespace foo and bar (expect failure only from legacy to foo)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				for _, from := range fromNamespaces {
					for _, to := range toNamespaces {
						url := fmt.Sprintf("http://httpbin.%s:8000/ip", to)
						if from == "legacy" && to == "foo" {
							app.AssertSleepPodRequestFailure(t, from, url)
						} else {
							app.AssertSleepPodRequestSuccess(t, from, url)
						}
					}
				}
			})
		})

		t.NewSubTest("mTLS enabled globally").Run(func(t test.TestHelper) {
			t.LogStep("Apply strict mTLS for the entire mesh")
			oc.ApplyString(t, meshNamespace, PeerAuthenticationMTLSStrict)
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, PeerAuthenticationMTLSStrict)
			})

			t.LogStep("Check connectivity from namespaces foo, bar, and legacy to namespace foo and bar (expect failure from legacy)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				for _, from := range fromNamespaces {
					for _, to := range toNamespaces {
						url := fmt.Sprintf("http://httpbin.%s:8000/ip", to)
						if from == "legacy" {
							app.AssertSleepPodRequestFailure(t, from, url)
						} else {
							app.AssertSleepPodRequestSuccess(t, from, url)
						}
					}
				}
			})
		})
	})
}

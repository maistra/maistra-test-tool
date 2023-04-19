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
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/common"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMTlsMigration(t *testing.T) {
	test.NewTest(t).Id("T19").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		meshNamespace := env.GetDefaultMeshNamespace()

		t.Cleanup(func() {
			oc.RecreateNamespace(t, "foo", "bar", "legacy") // TODO: recreate all three namespaces with a single call to RecreateNamespace
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and sleep in multiple namespaces")
		app.InstallAndWaitReady(t,
			app.Httpbin("foo"),
			app.Httpbin("bar"),
			app.Sleep("foo"),
			app.Sleep("bar"),
			app.SleepNoSidecar("legacy"))

		fromNamespaces := []string{"foo", "bar", "legacy"}
		toNamespaces := []string{"foo", "bar"}

		t.LogStep("Check connectivity from namespaces foo, bar, and legacy to namespace foo and bar")
		for _, from := range fromNamespaces {
			for _, to := range toNamespaces {
				assertConnectionSuccessful(t, from, to)
			}
		}

		t.NewSubTest("mTLS enabled in foo").Run(func(t test.TestHelper) {
			t.LogStep("Apply strict mTLS in namespace foo")
			oc.ApplyString(t, "foo", PeerAuthenticationMTLSStrict)

			t.LogStep("Check connectivity from namespaces foo, bar, and legacy to namespace foo and bar (expect failure only from legacy to foo)")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				for _, from := range fromNamespaces {
					for _, to := range toNamespaces {
						if from == "legacy" && to == "foo" {
							assertConnectionFailure(t, from, to)
						} else {
							assertConnectionSuccessful(t, from, to)
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
						if from == "legacy" {
							assertConnectionFailure(t, from, to)
						} else {
							assertConnectionSuccessful(t, from, to)
						}
					}
				}
			})
		})
	})
}

func assertConnectionSuccessful(t test.TestHelper, from string, to string) {
	curlFromTo(t, from, to,
		assert.OutputContains("200",
			fmt.Sprintf("%s connects to %s", from, to),
			fmt.Sprintf("%s can't connect to %s", from, to)))
}

func assertConnectionFailure(t test.TestHelper, from string, to string) {
	curlFromTo(t, from, to,
		assert.OutputContains("failed to connect",
			fmt.Sprintf("%s can't conect to %s", from, to),
			fmt.Sprintf("%s can connect to %s, but shouldn't", from, to)))
}

func curlFromTo(t test.TestHelper, from string, to string, checks ...common.CheckFunc) {
	oc.Exec(t,
		pod.MatchingSelector("app=sleep", from),
		"sleep",
		fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}" || echo "failed to connect"`, to, from, to),
		checks...)
}

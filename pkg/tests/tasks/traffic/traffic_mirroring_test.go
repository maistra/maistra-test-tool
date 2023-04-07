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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	. "github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestMirroring(t *testing.T) {
	NewTest(t).Id("T7").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t,
			app.HttpbinV1(ns),
			app.HttpbinV2(ns),
			app.Sleep(ns))

		t.NewSubTest("no mirroring").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, httpbinAllv1)

			t.LogStep("sending HTTP request from sleep to httpbin-v1, not expecting mirroring to v2")
			retry.UntilSuccess(t, func(t TestHelper) {
				nonce := NewNonce()

				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns),
					"sleep",
					"curl -sS http://httpbin:8000/headers?nonce="+nonce)

				oc.Logs(t,
					pod.MatchingSelector("app=httpbin,version=v1", ns),
					"httpbin",
					assert.OutputContains(
						"GET /headers?nonce="+nonce,
						"request received by httpbin-v1",
						"request not received by httpbin-v1"))

				oc.Logs(t,
					pod.MatchingSelector("app=httpbin,version=v2", ns),
					"httpbin",
					assert.OutputDoesNotContain(
						"GET /headers?nonce="+nonce,
						"request not mirrored to httpbin-v2",
						"request mirrored to httpbin-v2 but shouldn't have been"))
			})
		})

		t.NewSubTest("mirroring to httpbin-v2").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, httpbinMirrorv2)

			t.LogStep("sending HTTP request from sleep to httpbin-v1, expecting mirroring to v2")
			retry.UntilSuccess(t, func(t TestHelper) {
				nonce := NewNonce()

				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns),
					"sleep",
					"curl -sS http://httpbin:8000/headers?nonce="+nonce)

				oc.Logs(t,
					pod.MatchingSelector("app=httpbin,version=v1", ns),
					"httpbin",
					assert.OutputContains(
						"GET /headers?nonce="+nonce,
						"request received by httpbin-v1",
						"request not received by httpbin-v1"))

				oc.Logs(t,
					pod.MatchingSelector("app=httpbin,version=v2", ns),
					"httpbin",
					assert.OutputContains(
						"GET /headers?nonce="+nonce,
						"request mirrored to httpbin-v2",
						"request not mirrored to httpbin-v2, but should have been"))
			})
		})
	})
}

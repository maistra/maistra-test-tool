// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ossm

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestInjectionInPrivelegedPods(t *testing.T) {
	NewTest(t).Groups(Full, Disconnected, ARM).Run(func(t TestHelper) {
		if env.GetOperatorVersion().LessThan(version.OPERATOR_2_6_2) {
			t.Skip("This test requires the operator version to be at least 2.6.2")
		}
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-8001")

		t.Cleanup(func() {
			app.Uninstall(t, app.Httpbin(ns.Foo))
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStep("Deploy smcp")
		DeployControlPlane(t)

		t.LogStep("Patch SMCP to enable mTLS in dataPlane and controlPlane")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  security:
    dataPlane:
      mtls: true
    controlPlane:
      mtls: true
`)

		t.LogStep("Deploy httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo))

		t.NewSubTest("Check sleep with explicitly defined SecurityContext with same uid/gid (1001)").Run(func(t TestHelper) {
			runSecurityContextTest(t, 1001, 1001, "uid=1002(1002) gid=1002 groups=1002")
		})

		t.NewSubTest("Check sleep with explicitly defined SecurityContext with root uid/gid").Run(func(t TestHelper) {
			runSecurityContextTest(t, 0, 0, "uid=1(bin) gid=1(bin) groups=1(bin)")
		})
	})
}

func runSecurityContextTest(t TestHelper, uid, gid int, expectedIDOutput string) {
	t.Cleanup(func() {
		app.Uninstall(t, app.SleepSecurityContext(ns.Foo, uid, gid))
	})

	t.LogStep("Provide privileged policy to sleep SA")
	shell.Execute(t, "oc adm policy add-scc-to-user privileged -z sleep -n foo")

	t.LogStep("Deploy sleep")
	app.InstallAndWaitReady(t,
		app.SleepSecurityContext(ns.Foo, uid, gid),
	)

	t.LogStepf("Verify that UID, GID and Groups were changed to: %s", expectedIDOutput)
	oc.Exec(t, pod.MatchingSelector("app=sleep", ns.Foo), "istio-proxy",
		"id",
		assert.OutputContains(
			expectedIDOutput,
			"UID, GID and Groups were changed",
			"UID, GID and Groups were not changed"))

	t.LogStep("Verify that a request from sleep to httpbin returns 200")
	app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/ip")
}

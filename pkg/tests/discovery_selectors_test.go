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

package ossm

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestConversionDiscoverySelectors(t *testing.T) {
	NewTest(t).Id("T41").Groups(Full, InterOp).Run(func(t TestHelper) {
		t.Log("This test checks for discoverySelectors to convert from helm values to smcp spec")

		ns := "bookinfo"

		ossm.DeployControlPlane(t)

		t.NewSubTest("discoverySelecotrs").Run(func(t TestHelper) {
			t.LogStep("Update SMCP spec.controlPlane.meshConfig.discoverySelectors: Enabled")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `{"spec":{"controlPlane":{"disoverySelectors"}}}}}`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.LogStep("Install httpbin and sleep pod")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Sleep(ns))
		t.Cleanup(func() {
			oc.Patch(t, meshNamespace,
				"smcp", smcpName,
				"json",
				`[{"op": "remove", "path": "/spec/controlPlane/discoverySelectors"}]`)
			app.Uninstall(t, app.Httpbin(ns), app.Sleep(ns))
		})

		t.NewSubTest("Helmtospec").Run(func(t TestHelper) {
			t.LogStep("Convert the Helm Values to smcp spec Values")
			oc.ApplyString(t, ns, DSEnabled)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, DSEnabled)
			})
			oc.ApplyString(t, ns, DSEnabled)
			t.LogStep("Verify a request to path /ip is allowed")
			assertHttpbinRequestSucceeds(t, ns, httpbinRequest("GET", "/ip"))

		})
	})
}

func assertHttpbinRequestSucceeds(t TestHelper, ns string) {
	t.LogStep("Check if the httpbin returns 200 OK")
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=httpbin", ns),
			"httpbin",
			`curl http://httpbin:8000/ip -s -o /dev/null -w "%{http_code}"`,
			assert.OutputContains(
				"200",
				"Got expected 200 OK from httpbin",
				"Expected 200 OK from httpbin, but got a different HTTP code"))
	})
}

const (
	DSEnabled = `
apiVersion: install.istio.io/v1alpha1
kind: ServiceMeshControlPlane
metadata:
  namespace: istio-system
spec:
	# You may override parts of meshconfig by uncommenting the following lines.
  meshConfig:
	discoverySelectors:
    - matchLabels:
		istio-discovery: enabled`
)

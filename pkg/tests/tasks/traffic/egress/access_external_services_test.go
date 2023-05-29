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
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAccessExternalServices(t *testing.T) {
	test.NewTest(t).Id("T11").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		smcpName := env.GetDefaultSMCPName()
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns.Bookinfo))
			oc.Patch(t,
				meshNamespace,
				"smcp", smcpName,
				"json", `[{"op": "remove", "path": "/spec/proxy"}]`)
		})

		t.Log("This test validates accesses to external services")

		ossm.DeployControlPlane(t)

		t.LogStepf("Install sleep into %s", ns.Bookinfo)
		sleep := app.Sleep(ns.Bookinfo)
		app.InstallAndWaitReady(t, sleep)

		t.LogStepf("Install httpbin in %s", ns.MeshExternal)
		httpbin := app.HttpbinNoSidecar(ns.MeshExternal)
		app.InstallAndWaitReady(t, httpbin)

		t.LogStep("Make request to external httpbin from sleep")
		httpbinHeadersUrl := fmt.Sprintf("http://%s.%s:8000/headers", httpbin.Name(), httpbin.Namespace())
		assertRequestSuccess(t, sleep, httpbinHeadersUrl)

		t.LogStep("Make sure that external httpbin was not discovered by Istio - it would happen if mesh-external namespaces was added to the SMMR")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			shell.Execute(t,
				fmt.Sprintf("istioctl pc endpoint deploy/sleep -n %s", ns.Bookinfo),
				assert.OutputDoesNotContain(
					fmt.Sprintf("%s.%s.svc.cluster.local", httpbin.Name(), httpbin.Namespace()),
					"Httpbin was not discovered",
					"Expected Httpbin to not be discovered, but it was."))
		})

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

		t.LogStep("Make request to external httpbin from sleep again, and expect it denied")
		assertRequestFailure(t, sleep, httpbinHeadersUrl)

		t.NewSubTest("allow request to external httpbin after applying ServiceEntry").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns.Bookinfo, httpbinServiceEntry)
			})

			t.LogStep("Apply a ServiceEntry for external httpbin")
			oc.ApplyString(t, ns.Bookinfo, httpbinServiceEntry)

			t.LogStep("Send a request to external httpbin")
			assertRequestSuccess(t, sleep, httpbinHeadersUrl)
		})
	})
}

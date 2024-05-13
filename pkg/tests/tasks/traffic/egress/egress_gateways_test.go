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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressGateways(t *testing.T) {
	NewTest(t).Id("T13").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {

		ns := "bookinfo"
		ns1 := "mesh-external"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("HTTP").Run(func(t TestHelper) {
			t.LogStepf("Install external httpbin")
			httpbin := app.HttpbinNoSidecar(ns1)
			app.InstallAndWaitReady(t, httpbin)

			t.LogStep("Apply a ServiceEntry for external httpbin")
			oc.ApplyString(t, ns, httpbinServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, httpbinServiceEntry)
			})

			t.LogStep("Apply a gateway and virtual service for external httpbin")
			oc.ApplyTemplate(t, ns, httpbinHttpGateway, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, httpbinHttpGateway, smcp)
			})

			app.AssertSleepPodRequestSuccess(t, ns, "http://httpbin.mesh-external:8000/headers")
		})

		t.NewSubTest("HTTPS").Run(func(t TestHelper) {
			t.LogStep("Install external nginx")
			app.InstallAndWaitReady(t, app.NginxExternalTLS(ns1))

			t.LogStep("Create ServiceEntry for external nginx, port 80 and 443")
			oc.ApplyString(t, meshNamespace, nginxServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, nginxServiceEntry)
			})

			t.LogStep("Create a TLS ServiceEntry to external nginx")
			oc.ApplyString(t, ns, nginxServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, nginxServiceEntry)
			})

			t.LogStep("Create a https Gateway to external nginx")
			oc.ApplyTemplate(t, ns, nginxTlsPassthroughGateway, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, nginxTlsPassthroughGateway, smcp)
			})

			t.Log("Send HTTPS request to external nginx")
			app.AssertSleepPodRequestSuccess(
				t,
				ns,
				"https://my-nginx.mesh-external.svc.cluster.local",
				app.CurlOpts{Options: []string{"--insecure"}},
			)
		})
	})
}

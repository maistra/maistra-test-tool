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
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressGateways(t *testing.T) {
	NewTest(t).Id("T13").Groups(Full, InterOp, ARM, Persistent).Run(func(t TestHelper) {
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
		})
		smcp := ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns.Bookinfo))

		t.NewSubTest("HTTP").Run(func(t TestHelper) {
			t.LogStepf("Install external httpbin")
			httpbin := app.HttpbinNoSidecar(ns.MeshExternal)
			app.InstallAndWaitReady(t, httpbin)

			t.LogStep("Apply a ServiceEntry for external httpbin")
			oc.ApplyString(t, ns.Bookinfo, httpbinServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns.Bookinfo, httpbinServiceEntry)
			})

			t.LogStep("Apply a gateway and virtual service for external httpbin")
			oc.ApplyTemplate(t, ns.Bookinfo, httpbinHttpGateway, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Bookinfo, httpbinHttpGateway, smcp)
			})

			app.AssertSleepPodRequestSuccess(t, ns.Bookinfo, "http://httpbin.mesh-external:8000/headers")
		})

		t.NewSubTest("HTTPS").Run(func(t TestHelper) {
			t.LogStep("Install external nginx")
			app.InstallAndWaitReady(t, app.NginxExternalTLS(ns.MeshExternal))

			t.LogStep("Create ServiceEntry for external nginx, port 80 and 443")
			oc.ApplyString(t, meshNamespace, nginxServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, nginxServiceEntry)
			})

			t.LogStep("Create a TLS ServiceEntry to external nginx")
			oc.ApplyString(t, ns.Bookinfo, nginxServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns.Bookinfo, nginxServiceEntry)
			})

			t.LogStep("Create a https Gateway to external nginx")
			oc.ApplyTemplate(t, ns.Bookinfo, nginxTlsPassthroughGateway, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Bookinfo, nginxTlsPassthroughGateway, smcp)
			})

			t.Log("Send HTTPS request to external nginx")
			app.AssertSleepPodRequestSuccess(
				t,
				ns.Bookinfo,
				"https://my-nginx.mesh-external.svc.cluster.local",
				app.CurlOpts{Options: []string{"--insecure"}},
			)
		})
	})
}

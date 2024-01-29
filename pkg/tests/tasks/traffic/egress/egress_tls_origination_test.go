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
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressTLSOrigination(t *testing.T) {
	test.NewTest(t).Id("T12").Groups(test.Full, test.InterOp, test.ARM).Run(func(t test.TestHelper) {
		sleep := app.Sleep(ns.Bookinfo)
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.MeshExternal)
			app.Uninstall(t, sleep)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, sleep)

		t.NewSubTest("TrafficManagement_egress_tls_origination").Run(func(t test.TestHelper) {
			t.Log("TLS origination for egress traffic")
			t.Cleanup(func() {
				app.Uninstall(t, app.NginxExternalTLS(ns.MeshExternal))
				oc.DeleteFromString(t, ns.Bookinfo, nginxServiceEntry)
				oc.DeleteFromString(t, ns.Bookinfo, meshRouteHttpRequestsToHttpsPort)
				oc.DeleteFromString(t, ns.Bookinfo, originateTlsToNginx)
			})

			app.InstallAndWaitReady(t, app.NginxExternalTLS(ns.MeshExternal))
			oc.ApplyString(t, ns.Bookinfo, nginxServiceEntry)
			oc.ApplyString(t, ns.Bookinfo, meshRouteHttpRequestsToHttpsPort)
			oc.ApplyString(t, ns.Bookinfo, originateTlsToNginx)

			assertRequestSuccess(t, sleep, "http://my-nginx.mesh-external.svc.cluster.local")
		})
	})
}

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
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressTLSOrigination(t *testing.T) {
	test.NewTest(t).Id("T12").Groups(test.Full, test.InterOp, test.ARM).Run(func(t test.TestHelper) {

		ns := "bookinfo"
		ns1 := "mesh-external"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns1)
			app.Uninstall(t, app.Sleep(ns))
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("TrafficManagement_egress_tls_origination").Run(func(t test.TestHelper) {
			t.Log("TLS origination for egress traffic")
			t.Cleanup(func() {
				app.Uninstall(t, app.NginxExternalTLS(ns1))
				oc.DeleteFromString(t, ns, nginxServiceEntry)
				oc.DeleteFromString(t, ns, meshRouteHttpRequestsToHttpsPort)
				oc.DeleteFromString(t, ns, originateTlsToNginx)
			})

			app.InstallAndWaitReady(t, app.NginxExternalTLS(ns1))
			oc.ApplyString(t, ns, nginxServiceEntry)
			oc.ApplyString(t, ns, meshRouteHttpRequestsToHttpsPort)
			oc.ApplyString(t, ns, originateTlsToNginx)

			app.AssertSleepPodRequestSuccess(t, ns, "http://my-nginx.mesh-external.svc.cluster.local")
		})
	})
}

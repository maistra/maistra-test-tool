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
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTLSOriginationSDS(t *testing.T) {
	NewTest(t).Id("T15").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("ServiceEntry").Run(func(t TestHelper) {
			t.LogStep("Perform TLS origination with an egress gateway")
			oc.ApplyString(t, ns, ExServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ExServiceEntry)
			})
			retry.UntilSuccess(t, func(t test.TestHelper) {
				execInSleepPod(t, ns,
					`curl -sSL -o /dev/null -D - http://istio.io`,
					assert.OutputContains(
						"301",
						"Expected 301 Moved Permanently",
						"Unexpected response, expected 301 Moved Permanently"))
			})

			t.LogStep("Create a Gateway to external istio.io")
			oc.ApplyTemplate(t, ns, ExGatewayTLSFileTemplate, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, ExGatewayTLSFileTemplate, smcp)
			})
			retry.UntilSuccess(t, func(t test.TestHelper) {
				execInSleepPod(t, ns,
					`curl -sSL -o /dev/null -D - http://istio.io`,
					assert.OutputContains(
						"HTTP/1.1 200 OK",
						"Expected 200 from istio.io",
						"Unexpected response, expected 200"))
			})
		})

		t.NewSubTest("Gateway").Run(func(t TestHelper) {
			t.Log("Perform mTLS origination with an egress gateway")
			nsNginx := "mesh-external"
			t.Cleanup(func() {
				oc.DeleteNamespace(t, nsNginx)
				oc.DeleteSecret(t, meshNamespace, nginxClientCertKey)
				oc.DeleteSecret(t, meshNamespace, nginxClientCert)
				oc.DeleteSecret(t, meshNamespace, nginxServerCACert)
				oc.DeleteFromTemplate(t, ns, EgressGatewaySDSTemplate, smcp)
				oc.DeleteFromString(t, meshNamespace, meshExternalServiceEntry)
				oc.DeleteFromString(t, meshNamespace, OriginateSDS)
			})

			t.LogStep("Deploy nginx mTLS server and create secrets in the mesh namespace")

			nginx := examples.Nginx{Namespace: nsNginx}
			nginx.Install_mTLS(env.GetRootDir() + "/testdata/examples/x86/nginx/nginx_mesh_external_ssl.conf")

			oc.CreateTLSSecretWithCACert(t, meshNamespace, "client-credential", nginxClientCertKey, nginxClientCert, nginxServerCACert)

			oc.ApplyTemplate(t, ns, EgressGatewaySDSTemplate, smcp)
			oc.ApplyString(t, meshNamespace, meshExternalServiceEntry)
			oc.ApplyString(t, meshNamespace, OriginateSDS)

			t.Log("Verify NGINX server")

			retry.UntilSuccess(t, func(t test.TestHelper) {
				execInSleepPod(t, ns,
					`curl -sS http://my-nginx.mesh-external.svc.cluster.local`,
					assert.OutputContains(
						"Welcome to nginx",
						"Success. Get expected response: Welcome to nginx",
						"ERROR: Expected Welcome to nginx; Got unexpected response"))
			})
		})
	})
}

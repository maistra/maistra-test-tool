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

func TestTLSOriginationSDS(t *testing.T) {
	NewTest(t).Id("T15").Groups(Full, InterOp, ARM, Persistent).Run(func(t TestHelper) {
		t.Log("Perform mTLS origination with an egress gateway")
		smcp := ossm.DeployControlPlane(t)

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
			oc.RecreateNamespace(t, ns.MeshExternal)
			oc.DeleteSecret(t, meshNamespace, "client-credential")
			oc.DeleteFromTemplate(t, ns.Bookinfo, nginxTlsIstioMutualGateway, smcp)
			oc.DeleteFromString(t, meshNamespace, nginxServiceEntry, originateMtlsSdsSToNginx)
		})

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns.Bookinfo))

		t.LogStep("Deploy nginx mTLS server and create secrets in the mesh namespace")
		app.InstallAndWaitReady(t, app.NginxExternalMTLS(ns.MeshExternal))
		oc.CreateGenericSecretFromFiles(t, meshNamespace, "client-credential",
			"tls.key="+nginxClientCertKey,
			"tls.crt="+nginxClientCert,
			"ca.crt="+nginxServerCACert)
		oc.ApplyTemplate(t, ns.Bookinfo, nginxTlsIstioMutualGateway, smcp)
		oc.ApplyString(t, meshNamespace, nginxServiceEntry, originateMtlsSdsSToNginx)

		t.Log("Send HTTP request to external nginx to verify mTLS origination")
		app.AssertSleepPodRequestSuccess(t, ns.Bookinfo, "http://my-nginx.mesh-external.svc.cluster.local")
	})
}

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

func TestTLSOriginationSDS(t *testing.T) {
	NewTest(t).Id("T15").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {

		ns := "bookinfo"
		ns1 := "mesh-external"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
			oc.RecreateNamespace(t, ns)
			oc.DeleteSecret(t, meshNamespace, "client-credential")
			oc.DeleteFromTemplate(t, ns, nginxTlsIstioMutualGateway, smcp)
			oc.DeleteFromString(t, meshNamespace, nginxServiceEntry, originateMtlsSdsSToNginx)
		})

		t.Log("Perform mTLS origination with an egress gateway")
		ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.LogStep("Deploy nginx mTLS server and create secrets in the mesh namespace")
		app.InstallAndWaitReady(t, app.NginxExternalMTLS(ns1))
		oc.CreateGenericSecretFromFiles(t, meshNamespace, "client-credential",
			"tls.key="+nginxClientCertKey,
			"tls.crt="+nginxClientCert,
			"ca.crt="+nginxServerCACert)
		oc.ApplyTemplate(t, ns, nginxTlsIstioMutualGateway, smcp)
		oc.ApplyString(t, meshNamespace, nginxServiceEntry, originateMtlsSdsSToNginx)

		t.Log("Send HTTP request to external nginx to verify mTLS origination")
		app.AssertSleepPodRequestSuccess(t, ns, "http://my-nginx.mesh-external.svc.cluster.local")
	})
}

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
	"github.com/maistra/maistra-test-tool/pkg/util/istioctl"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTLSOrigination(t *testing.T) {
	NewTest(t).Id("T14").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		t.Log("This test verifies that TLS origination works in 2 scenarios:")
		t.Log("  1) Egress gateway TLS Origination")
		t.Log("  2) MTLS Origination with file mount (certificates mounted in egress gateway pod)")

		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns.Bookinfo))
		})

		smcp := ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns.Bookinfo))

		t.NewSubTest("Egress Gateway without file mount").Run(func(t TestHelper) {
			t.Log("Perform TLS origination with an egress gateway")

			t.LogStep("Install external nginx")
			app.InstallAndWaitReady(t, app.NginxExternalTLS(ns.MeshExternal))

			t.LogStep("Make sure that mesh external namespace is not discovered by Istio - it would happen if mesh-external namespaces was added to the SMMR")
			istioctl.CheckClusters(t,
				pod.MatchingSelector("app=sleep", ns.Bookinfo),
				assert.OutputDoesNotContain(
					fmt.Sprintf("%s.svc.cluster.local", ns.MeshExternal),
					"mesh-external namespace was not discovered",
					"Expected mesh-external to not be discovered, but it was."))

			t.LogStep("Create ServiceEntry for external nginx, port 80 and 443")
			oc.ApplyString(t, meshNamespace, nginxServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, nginxServiceEntry)
			})

			t.LogStep("Create a Gateway, DestinationRule, and VirtualService to route requests to external nginx through the egress gateway")
			oc.ApplyTemplate(t, ns.Bookinfo, nginxTlsIstioMutualGateway, smcp)
			oc.ApplyString(t, meshNamespace, originateTlsToNginx)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Bookinfo, nginxTlsIstioMutualGateway, smcp)
				oc.DeleteFromString(t, meshNamespace, originateTlsToNginx)
			})

			t.LogStep("Verify that request to external nginx is routed through the egress gateway (response 200 indicates that the TLS origination is done by the egress gateway)")
			app.ExecInSleepPod(t, ns.Bookinfo,
				`curl -sS http://my-nginx.mesh-external.svc.cluster.local`,
				assert.OutputContains(
					"Welcome to nginx",
					"Get expected response: Welcome to nginx",
					"Expected Welcome to nginx; Got unexpected response"))
		})

		t.NewSubTest("mTLS with file mount").Run(func(t TestHelper) {
			t.Log("Perform mTLS origination with an egress gateway")
			t.Cleanup(func() {
				app.Uninstall(t, app.NginxExternalMTLS(ns.MeshExternal))
				oc.DeleteSecret(t, meshNamespace, "nginx-client-certs", "nginx-ca-certs")
				oc.DeleteFromTemplate(t, ns.Bookinfo, nginxTlsIstioMutualGateway, smcp)
				oc.DeleteFromString(t, meshNamespace, nginxServiceEntry, originateMtlsToNginx)
				// revert patch to istio-egressgateway
				oc.TouchSMCP(t, meshNamespace, smcp.Name)
				// TODO: this is a potential bug; investigate why the following is necessary
				// ingressgateway needs to be restarted or it will continue reporting the following error:
				// error	cache	resource:file-root:/etc/istio/nginx-ca-certs/example.com.crt failed to generate secret for proxy from file: open /etc/istio/nginx-ca-certs/example.com.crt: no such file or directory
				oc.DeletePod(t, pod.MatchingSelector(fmt.Sprintf("app=istio-ingressgateway,maistra-control-plane=%s", meshNamespace), meshNamespace))
			})

			t.LogStep("Deploy nginx mTLS server and create secrets in the mesh namespace")
			oc.CreateTLSSecret(t, meshNamespace, "nginx-client-certs", nginxClientCertKey, nginxClientCert)
			oc.CreateGenericSecretFromFiles(t, meshNamespace,
				"nginx-ca-certs",
				"example.com.crt="+nginxServerCACert)
			app.Install(t, app.NginxExternalMTLS(ns.MeshExternal))

			t.LogStep("Patch egress gateway with File Mount configuration")
			oc.Patch(t, meshNamespace, "deploy", "istio-egressgateway", "json", gatewayPatchAdd)

			t.LogStep("Configure mTLS origination for egress traffic")
			oc.ApplyTemplate(t, ns.Bookinfo, nginxTlsIstioMutualGateway, smcp)
			oc.ApplyString(t, meshNamespace, nginxServiceEntry, originateMtlsToNginx)

			t.LogStep("Wait for egress gateway and nginx to be ready")
			oc.WaitDeploymentRolloutComplete(t, meshNamespace, "istio-egressgateway")
			app.WaitReady(t, app.NginxExternalMTLS(ns.MeshExternal))

			t.LogStep("Verify NGINX server")
			app.ExecInSleepPod(t, ns.Bookinfo,
				`curl -sS http://my-nginx.mesh-external.svc.cluster.local`,
				assert.OutputContains(
					"Welcome to nginx",
					"Get expected response: Welcome to nginx",
					"Expected Welcome to nginx; Got unexpected response"))
		})
	})
}

var (
	gatewayPatchAdd = `[
    {
        "op": "add",
        "path": "/spec/template/spec/containers/0/volumeMounts/0",
        "value": {
            "mountPath": "/etc/istio/nginx-client-certs",
            "name": "nginx-client-certs",
            "readOnly": true
        }
    },
    {
        "op": "add",
        "path": "/spec/template/spec/volumes/0",
        "value": {
            "name": "nginx-client-certs",
            "secret": {
                "secretName": "nginx-client-certs",
                "optional": true
            }
        }
    },
    {
        "op": "add",
        "path": "/spec/template/spec/containers/0/volumeMounts/1",
        "value": {
            "mountPath": "/etc/istio/nginx-ca-certs",
            "name": "nginx-ca-certs",
            "readOnly": true
        }
    },
    {
        "op": "add",
        "path": "/spec/template/spec/volumes/1",
        "value": {
            "name": "nginx-ca-certs",
            "secret": {
                "secretName": "nginx-ca-certs",
                "optional": true
            }
        }
    }
]`
)

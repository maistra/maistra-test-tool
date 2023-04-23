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
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTLSOrigination(t *testing.T) {
	NewTest(t).Id("T14").Groups(Full, InterOp).Run(func(t TestHelper) {
		t.Log("This test verifies that TLS origination works in 2 scenarios:")
		t.Log("  1) Egress gateway TLS Origination")
		t.Log("  2) MTLS Origination with file mount (certificates mounted in egress gateway pod)")

		ns := "bookinfo"
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("Egress Gateway without file mount").Run(func(t TestHelper) {
			t.Log("Perform TLS origination with an egress gateway")

			t.LogStep("Create ServiceEntry for istio.io, port 80 and 443")
			oc.ApplyString(t, ns, ExServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ExServiceEntry)
			})

			t.LogStep("Verify that the egress gateway is working: expect 301 Moved Permanently from istio.io")
			execInSleepPod(t, ns,
				`curl -sSL -o /dev/null -D - http://istio.io`,
				assert.OutputContains(
					"301",
					"Got expected 301 Moved Permanently",
					"Not expected response, expected 301 Moved Permanently"))

			t.LogStep("Create a Gateway, DestinationRule, and VirtualService to route requests to istio.io")
			oc.ApplyTemplate(t, ns, ExGatewayTLSFileTemplate, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, ExGatewayTLSFileTemplate, smcp)
			})

			t.LogStep("Verify that request to http://istio.io is routed through the egress gateway (response 200 indicates that the TLS origination is done by the egress gateway)")
			execInSleepPod(t, ns,
				fmt.Sprintf(`curl -sSL -o /dev/null %s -w "%%{http_code}" %s`, getCurlProxyParams(t), "http://istio.io"),
				assert.OutputContains("200",
					"Got expected 200 response",
					"Unexpected response from http://istio.io"))
		})

		t.NewSubTest("mTLS with file mount").Run(func(t TestHelper) {
			t.Log("Perform mTLS origination with an egress gateway")
			nsNginx := "mesh-external"
			t.Cleanup(func() {
				app.Uninstall(t, app.NginxWithMTLS(nsNginx))
				oc.DeleteSecret(t, meshNamespace, "nginx-client-certs", "nginx-ca-certs")
				oc.DeleteFromTemplate(t, ns, nginxGatewayTLSTemplate, smcp)
				oc.DeleteFromString(t, meshNamespace, meshExternalServiceEntry, nginxMeshRule)
				// revert patch to istio-egressgateway
				oc.TouchSMCP(t, meshNamespace, smcp.Name)
				// TODO: this is a potential bug; investigate why the following is necessary
				// ingressgateway needs to be restarted or it will continue reporting the following error:
				// error	cache	resource:file-root:/etc/istio/nginx-ca-certs/example.com.crt failed to generate secret for proxy from file: open /etc/istio/nginx-ca-certs/example.com.crt: no such file or directory
				oc.DeletePod(t, pod.MatchingSelector("app=istio-ingressgateway", meshNamespace))
			})

			t.LogStep("Deploy nginx mTLS server and create secrets in the mesh namespace")
			oc.CreateTLSSecret(t, meshNamespace, "nginx-client-certs", nginxClientCertKey, nginxClientCert)
			oc.CreateGenericSecretFromFiles(t, meshNamespace,
				"nginx-ca-certs",
				"example.com.crt="+nginxServerCACert)
			app.Install(t, app.NginxWithMTLS(nsNginx))

			t.LogStep("Patch egress gateway with File Mount configuration")
			oc.Patch(t, meshNamespace, "deploy", "istio-egressgateway", "json", gatewayPatchAdd)

			t.LogStep("Configure MTLS origination for egress traffic")
			oc.ApplyTemplate(t, ns, nginxGatewayTLSTemplate, smcp)
			oc.ApplyString(t, meshNamespace, meshExternalServiceEntry, nginxMeshRule)

			t.LogStep("Wait for egress gateway and nginx to be ready")
			oc.WaitDeploymentRolloutComplete(t, meshNamespace, "istio-egressgateway")
			app.WaitReady(t, app.NginxWithMTLS(nsNginx))

			t.LogStep("Verify NGINX server")
			execInSleepPod(t, ns,
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

	nginxGatewayTLSTemplate = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: istio-egressgateway
spec:
  selector:
    istio: egressgateway
  servers:
  - port:
      number: 443
      name: https
      protocol: HTTPS
    hosts:
    - my-nginx.mesh-external.svc.cluster.local
    tls:
      mode: ISTIO_MUTUAL
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: egressgateway-for-nginx
spec:
  host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
  subsets:
  - name: nginx
    trafficPolicy:
      loadBalancer:
        simple: ROUND_ROBIN
      portLevelSettings:
      - port:
          number: 443
        tls:
          mode: ISTIO_MUTUAL
          sni: my-nginx.mesh-external.svc.cluster.local
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: direct-nginx-through-egress-gateway
spec:
  hosts:
  - my-nginx.mesh-external.svc.cluster.local
  gateways:
  - istio-egressgateway
  - mesh
  http:
  - match:
    - gateways:
      - mesh
      port: 80
    route:
    - destination:
        host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
        subset: nginx
        port:
          number: 443
      weight: 100
  - match:
    - gateways:
      - istio-egressgateway
      port: 443
    route:
    - destination:
        host: my-nginx.mesh-external.svc.cluster.local
        port:
          number: 443
      weight: 100
`

	nginxMeshRule = `
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: originate-mtls-for-nginx
spec:
  host: my-nginx.mesh-external.svc.cluster.local
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
    portLevelSettings:
    - port:
        number: 443
      tls:
        mode: MUTUAL
        clientCertificate: /etc/istio/nginx-client-certs/tls.crt
        privateKey: /etc/istio/nginx-client-certs/tls.key
        caCertificates: /etc/istio/nginx-ca-certs/example.com.crt
        sni: my-nginx.mesh-external.svc.cluster.local
`
)

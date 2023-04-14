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
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTLSOrigination(t *testing.T) {
	NewTest(t).Id("T14").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		t.Log("This test verifies that TLS origination works in 2 scenarios: 1) Egress gateway TLS Origination 2) MTLS Origination with file mount (certificates mounted in egress gateway pod)")
		ns := "bookinfo"
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
		})
		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("Egress Gateway without file mount").Run(func(t TestHelper) {
			t.Log("Perform TLS origination with an egress gateway")
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, ExGatewayTLSFileTemplate, smcp)
				oc.DeleteFromString(t, ns, ExServiceEntry)
			})

			t.LogStep("Create ServiceEntry for port 80 and 443 and verify that the egress gateway is working: expect 301 Moved Permanently from istio.io")
			oc.ApplyString(t, ns, ExServiceEntry)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns),
					"sleep", `curl -sSL -o /dev/null -D - http://istio.io`,
					assert.OutputContains(
						"301",
						"Expected 301 Moved Permanently",
						"ERROR: Not expected response, expected 301 Moved Permanently"))
			})

			t.LogStep("Create a Gateway to external istio.io and verify that the egress gateway is working: expect Get http://istio.io response 200")
			t.Log("Create a Gateway for ingress traffic, DestinationRule, and VirtualService to route to the external service")
			oc.ApplyTemplate(t, ns, ExGatewayTLSFileTemplate, smcp)
			t.Log("Expect Get http://istio.io response 200 because the TLS origination is done by the egress gateway")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				execInSleepPod(t, ns,
					fmt.Sprintf(`curl -sSL -o /dev/null %s -w "%%{http_code}" %s`, getCurlProxyParams(), "http://istio.io"),
					assert.OutputContains("200",
						"Got expected 200 response",
						"Unexpected response from http://istio.io"))
			})
		})

		t.NewSubTest("MTLS with file mount").Run(func(t TestHelper) {
			t.Log("Perform MTLS origination with an egress gateway")
			nsNginx := "mesh-external"
			t.Cleanup(func() {
				app.Uninstall(t, app.NginxWithMTLS(nsNginx))
				oc.DeleteSecret(t, meshNamespace, "nginx-client-certs")
				oc.DeleteSecret(t, meshNamespace, "nginx-ca-certs")
				// Rollout to istio-egressgateway to revert patch
				shell.Executef(t, `kubectl -n %s rollout undo deploy istio-egressgateway`, meshNamespace)
				oc.DeleteFromTemplate(t, ns, nginxGatewayTLSTemplate, smcp)
				oc.DeleteFromString(t, meshNamespace, meshExternalServiceEntry)
				oc.DeleteFromString(t, meshNamespace, nginxMeshRule)
			})

			t.LogStep("Deploy nginx mtls server and create secrets in the mesh namespace")
			app.InstallAndWaitReady(t, app.NginxWithMTLS(nsNginx))
			oc.CreateTLSSecret(t, meshNamespace, "nginx-client-certs", nginxClientCertKey, nginxClientCert)
			oc.CreateGenericSecretFromFiles(t, meshNamespace,
				"nginx-ca-certs",
				"example.com.crt="+nginxServerCACert)

			t.LogStep("Patch egress gateway with File Mount configuration")
			oc.Patch(t, meshNamespace, "deploy", "istio-egressgateway", "json", gatewayPatchAdd)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			// It's needed to verify that the egress gateway have all the files after the patch and check the rollout history?

			t.LogStep("Configure MTLS origination for egress traffic")
			oc.ApplyTemplate(t, ns, nginxGatewayTLSTemplate, smcp)
			oc.ApplyString(t, meshNamespace, meshExternalServiceEntry)
			oc.ApplyString(t, meshNamespace, nginxMeshRule)

			t.LogStep("Verify NGINX server")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", ns),
					"sleep",
					`curl -sS http://my-nginx.mesh-external.svc.cluster.local`,
					assert.OutputContains(
						"Welcome to nginx",
						"Success. Get expected response: Welcome to nginx",
						"ERROR: Expected Welcome to nginx; Got unexpected response"))
			})
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

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
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressGateways(t *testing.T) {
	NewTest(t).Id("T13").Groups(Full, InterOp).Run(func(t TestHelper) {
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Bookinfo)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install sleep pod")
		sleep := app.Sleep(ns.Bookinfo)
		app.InstallAndWaitReady(t, sleep)

		t.NewSubTest("HTTP").Run(func(t TestHelper) {
			t.LogStepf("Install external httpbin")
			httpbin := app.HttpbinNoSidecar(ns.MeshExternal)
			app.InstallAndWaitReady(t, httpbin)

			t.LogStep("Apply a ServiceEntry for external httpbin")
			httpbinValues := map[string]string{
				"Name":      httpbin.Name(),
				"Namespace": httpbin.Namespace(),
			}
			oc.ApplyTemplate(t, ns.Bookinfo, httpbinExt, httpbinValues)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Bookinfo, httpbinExt, httpbinValues)
			})

			t.LogStep("Apply a gateway and virtual service for external httpbin")
			oc.ApplyTemplate(t, ns.Bookinfo, externalHttpbinHttpGateway, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Bookinfo, externalHttpbinHttpGateway, smcp)
			})

			assertRequestSuccess(t, sleep, "http://httpbin.mesh-external:8000/headers")
		})

		t.NewSubTest("HTTPS").Run(func(t TestHelper) {
			t.LogStep("Install external nginx")
			app.InstallAndWaitReady(t, app.NginxExternalTLS(ns.MeshExternal))

			t.LogStep("Create ServiceEntry for external nginx, port 80 and 443")
			oc.ApplyString(t, meshNamespace, meshExternalNginx)
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, meshExternalNginx)
			})

			t.LogStep("Create a TLS ServiceEntry to external nginx")
			oc.ApplyString(t, ns.Bookinfo, meshExternalNginx)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns.Bookinfo, meshExternalNginx)
			})

			t.LogStep("Create a https Gateway to external nginx")
			oc.ApplyTemplate(t, ns.Bookinfo, externalNginxTLSPassthroughGateway, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns.Bookinfo, externalNginxTLSPassthroughGateway, smcp)
			})

			t.Log("Send HTTPS request to external nginx")
			assertInsecureRequestSuccess(t, sleep, "https://my-nginx.mesh-external.svc.cluster.local")
		})
	})
}

func assertInsecureRequestSuccess(t TestHelper, client app.App, url string) {
	url = fmt.Sprintf(`curl -sSL --insecure -o /dev/null -w "%%{http_code}" %s 2>/dev/null || echo %s`, url, curlFailedMessage)
	execInSleepPod(t, client.Namespace(), url,
		assert.OutputContains("200",
			fmt.Sprintf("Got expected 200 OK from %s", url),
			fmt.Sprintf("Expect 200 OK from %s, but got a different HTTP code", url)))
}

const (
	externalHttpbinHttpGateway = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: istio-egressgateway
spec:
  selector:
    istio: egressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - httpbin.mesh-external.svc.cluster.local
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: egress-gateway-route-egress-traffic-to-external-httpbin
spec:
  hosts:
  - httpbin.mesh-external.svc.cluster.local
  gateways:
  - istio-egressgateway
  http:
  - match:
    - port: 80
    route:
    - destination:
        host: httpbin.mesh-external.svc.cluster.local
        port:
          number: 80
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: mesh-route-egress-requests-to-external-httpbin-through-egress-gateway
spec:
  hosts:
  - httpbin.mesh-external.svc.cluster.local
  gateways:
  - mesh
  http:
  - match:
    - port: 80
    route:
    - destination:
        host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
        port:
          number: 80
`

	externalNginxTLSPassthroughGateway = `
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
      name: tls
      protocol: TLS
    hosts:
    - my-nginx.mesh-external.svc.cluster.local
    tls:
      mode: PASSTHROUGH
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: egress-gateway-route-egress-traffic-to-external-nginx
spec:
  hosts:
  - my-nginx.mesh-external.svc.cluster.local
  gateways:
  - istio-egressgateway
  tls:
  - match:
    - port: 443
      sniHosts:
      - my-nginx.mesh-external.svc.cluster.local
    route:
    - destination:
        host: my-nginx.mesh-external.svc.cluster.local
        port:
          number: 443
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: mesh-route-egress-traffic-to-external-nginx-through-egress-gateway
spec:
  hosts:
  - my-nginx.mesh-external.svc.cluster.local
  gateways:
  - mesh
  tls:
  - match:
    - port: 443
      sniHosts:
      - my-nginx.mesh-external.svc.cluster.local
    route:
    - destination:
        host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
        port:
          number: 443

`
)

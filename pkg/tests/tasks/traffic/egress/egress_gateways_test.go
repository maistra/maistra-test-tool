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
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressGateways(t *testing.T) {
	NewTest(t).Id("T13").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Sleep(ns))

		t.NewSubTest("HTTP").Run(func(t TestHelper) {
			t.LogStep("Create a ServiceEntry to external istio.io")
			oc.ApplyString(t, ns, ExServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ExServiceEntry)
			})

			assertRequestSuccess := func(url string) {
				t.LogStepf("Confirm that request to %s is successful", url)
				execInSleepPod(t, ns,
					fmt.Sprintf(`curl -sSL -o /dev/null %s -w "%%{http_code}" %s`, getCurlProxyParams(), url),
					assert.OutputContains("200",
						fmt.Sprintf("Got %s response", url),
						fmt.Sprintf("Unexpected response from %s", url)))
			}
			assertRequestFailure := func(url string) {
				t.LogStepf("Confirm that request to %s fails", url)
				execInSleepPod(t, ns,
					fmt.Sprintf(`curl -sSL -o /dev/null %s -w "%%{http_code}" %s`, getCurlProxyParams(), url),
					assert.OutputContains("503",
						fmt.Sprintf("Got %s failure", url),
						fmt.Sprintf("Unexpected response from %s", url)))
			}

			assertRequestSuccess("http://istio.io")

			t.LogStep("Create a Gateway to external istio.io")
			oc.ApplyTemplate(t, ns, ExGatewayTemplate, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, ExGatewayTemplate, smcp)
			})

			t.LogStep("Scale istio-egressgateway to zero to confirm that requests to istio.io are routed through it")
			oc.ScaleDeploymentAndWait(t, meshNamespace, "istio-egressgateway", 0)
			assertRequestFailure("http://istio.io")

			t.LogStep("Scale istio-egressgateway back to one to confirm that requests to istio.io are successful")
			oc.ScaleDeploymentAndWait(t, meshNamespace, "istio-egressgateway", 1)
			assertRequestSuccess("http://istio.io")
		})

		t.NewSubTest("HTTPS").Run(func(t TestHelper) {
			t.LogStep("Create a TLS ServiceEntry to external istio.io")
			oc.ApplyString(t, ns, ExServiceEntryTLS)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ExServiceEntryTLS)
			})

			assertRequestSuccess := func(url string) {
				t.LogStepf("Confirm that request to %s is successful", url)
				execInSleepPod(t, ns,
					`curl -sSL -o /dev/null -w "%{http_code}" `+url,
					assert.OutputContains("200",
						fmt.Sprintf("Got %s response", url),
						fmt.Sprintf("Unexpected response from %s", url)))
			}

			assertRequestFailure := func(url string) {
				t.LogStepf("Confirm that request to %s fails", url)
				execInSleepPod(t, ns,
					fmt.Sprintf(`curl -sSL -o /dev/null -w "%%{http_code}" %s || echo "connection failed"`, url),
					assert.OutputContains("connection failed",
						fmt.Sprintf("Got %s failure", url),
						fmt.Sprintf("Unexpected response from %s", url)))
			}

			assertRequestSuccess("https://istio.io")

			t.LogStep("Create a https Gateway to external istio.io")
			oc.ApplyTemplate(t, ns, ExGatewayHTTPSTemplate, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, ExGatewayHTTPSTemplate, smcp)
			})

			t.LogStep("Scale istio-egressgateway to zero to confirm that requests to istio.io are routed through it")
			oc.ScaleDeploymentAndWait(t, meshNamespace, "istio-egressgateway", 0)
			assertRequestFailure("https://istio.io")

			t.LogStep("Scale istio-egressgateway back to one to confirm that requests to istio.io are successful")
			oc.ScaleDeploymentAndWait(t, meshNamespace, "istio-egressgateway", 1)
			assertRequestSuccess("https://istio.io")
		})
	})
}

const (
	ExServiceEntryTLS = `
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: istio-io
spec:
  hosts:
  - istio.io
  ports:
  - number: 443
    name: tls
    protocol: TLS
  resolution: DNS
`

	ExGatewayTemplate = `
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
    - istio.io
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: egressgateway-for-istio-io
spec:
  host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
  subsets:
  - name: istio-io
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: direct-istio-io-through-egress-gateway
spec:
  hosts:
  - istio.io
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
        subset: istio-io
        port:
          number: 80
      weight: 100
  - match:
    - gateways:
      - istio-egressgateway
      port: 80
    route:
    - destination:
        host: istio.io
        port:
          number: 80
      weight: 100
`

	ExGatewayHTTPSTemplate = `
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
    - istio.io
    tls:
      mode: PASSTHROUGH
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: egressgateway-for-istio-io
spec:
  host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
  subsets:
  - name: istio-io
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: direct-istio-io-through-egress-gateway
spec:
  hosts:
  - istio.io
  gateways:
  - mesh
  - istio-egressgateway
  tls:
  - match:
    - gateways:
      - mesh
      port: 443
      sniHosts:
      - istio.io
    route:
    - destination:
        host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
        subset: istio-io
        port:
          number: 443
  - match:
    - gateways:
      - istio-egressgateway
      port: 443
      sniHosts:
      - istio.io
    route:
    - destination:
        host: istio.io
        port:
          number: 443
      weight: 100
`
)

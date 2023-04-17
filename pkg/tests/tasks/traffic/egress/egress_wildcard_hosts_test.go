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
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressWildcard(t *testing.T) {
	NewTest(t).Id("T16").Groups(Full, InterOp).Run(func(t TestHelper) {
		t.Log("This test checks if the wildcard in the ServiceEntry and Gateway works as expected for Egress traffic.")

		ns := "bookinfo"

		t.LogStep("Install the sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns))
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
		})

		t.NewSubTest("ServiceEntry").Run(func(t TestHelper) {
			t.LogStep("Configure ServiceEntry with wildcard host *.wikipedia.org")
			oc.ApplyString(t, ns, EgressWildcardServiceEntry)
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, EgressWildcardServiceEntry)
			})

			assertExternalRequestSuccess(t, ns)
		})

		t.NewSubTest("Gateway").Run(func(t TestHelper) {
			t.LogStep("Configure egress Gateway with wildcard host *.wikipedia.org")
			oc.ApplyTemplate(t, ns, EgressWildcardGatewayTemplate, smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, ns, EgressWildcardGatewayTemplate, smcp)
			})

			assertExternalRequestSuccess(t, ns)
		})
	})
}

func assertExternalRequestSuccess(t TestHelper, ns string) {
	t.LogStep("Check external request to en.wikipedia.org and de.wikipedia.org")
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			`curl -s https://en.wikipedia.org/wiki/Main_Page`,
			assert.OutputContains(
				"<title>Wikipedia, the free encyclopedia</title>",
				"Received the correct response from en.wikipedia.org",
				"Failed to receive the correct response from en.wikipedia.org"))

		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			`curl -s https://de.wikipedia.org/wiki/Wikipedia:Hauptseite`,
			assert.OutputContains(
				"<title>Wikipedia – Die freie Enzyklopädie</title>",
				"Received the correct response from de.wikipedia.org",
				"Failed to receive the correct response from de.wikipedia.org"))
	})
}

const (
	EgressWildcardServiceEntry = `
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: wikipedia
spec:
  hosts:
  - "*.wikipedia.org"
  ports:
  - number: 443
    name: https
    protocol: HTTPS
`
	EgressWildcardGatewayTemplate = `
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
    - "*.wikipedia.org"
    tls:
      mode: PASSTHROUGH
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: egressgateway-for-wikipedia
spec:
  host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
  subsets:
    - name: wikipedia
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: direct-wikipedia-through-egress-gateway
spec:
  hosts:
  - "*.wikipedia.org"
  gateways:
  - mesh
  - istio-egressgateway
  tls:
  - match:
    - gateways:
      - mesh
      port: 443
      sniHosts:
      - "*.wikipedia.org"
    route:
    - destination:
        host: istio-egressgateway.{{ .Namespace }}.svc.cluster.local
        subset: wikipedia
        port:
          number: 443
      weight: 100
  - match:
    - gateways:
      - istio-egressgateway
      port: 443
      sniHosts:
      - "*.wikipedia.org"
    route:
    - destination:
        host: www.wikipedia.org
        port:
          number: 443
      weight: 100
---
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: www-wikipedia
spec:
  hosts:
  - www.wikipedia.org
  ports:
  - number: 443
    name: https
    protocol: HTTPS
  resolution: DNS
`
)

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
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEgressWildcard(t *testing.T) {
	t.NewTest(t).Id("T16").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "bookinfo"

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})
	
	t.Log("This Test recieves the sleep pod name")
	app.InstallAndWaitReady(t, app.GetPodName(ns), app.Sleep(ns))
	t.LogStep("Recieve the sleep pod name")
	t.GetPodName(t, func(t TestHelper) {
		oc.Exec(t,
		pod.MatchingSelector("app=sleep", ns),
		"sleep",
		smcp,
		assert.OutputContains(
			"",
			"Successfully got the sleep pod name",
			"Failed to get the sleep pod name"),
	)},
)},

	t.NewSubTest("TrafficManagement_egress_direct_traffic_wildcard_host").Run(func(t TestHelper) {
		t.LogStep("Configure direct traffic to a wildcard host")
		oc.ApplyString(t, ns, EgressWildcardEntry, smcp)

		command := `curl -s https://en.wikipedia.org/wiki/Main_Page | grep -o "<title>.*</title>"; curl -s https://de.wikipedia.org/wiki/Wikipedia:Hauptseite | grep -o "<title>.*</title>"`

		func assertEgressWildcardEntry(t TestHelper, ns string, command string) {
			retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			ns,
			command,
			assert.OutputContains(
				"<title>Wikipedia, the free encyclopedia</title>\n<title>Wikipedia – Die freie Enzyklopädie</title>",
				"Successful. Recieved the correct Wikipedia response",
				"Error. Failed to recieve the correct Wikipedia response")
		)}
	)}
})
			

	t.NewSubTest("TrafficManagement_egress_gateway_wildcard_host").Run(func(t TestHelper) {
		t.LogStep("Configure egress gateway to a wildcard host")
		oc.ApplyString(t, ns, EgressWildcardGatewayTemplate, smcp)

		command := `curl -s https://en.wikipedia.org/wiki/Main_Page | grep -o "<title>.*</title>"; curl -s https://de.wikipedia.org/wiki/Wikipedia:Hauptseite | grep -o "<title>.*</title>"`

		func assertEgressWildcardGatewayTemplate(t TestHelper, ns string, command string) {
			retry.UntilSuccess(t, func(t test.TestHelper) {
				oc.Exec(t,
				pod.MatchingSelector("app=sleep", ns),
				ns,
				command,
				assert.OutputContains(
					"<title>Wikipedia, the free encyclopedia</title>\n<title>Wikipedia – Die freie Enzyklopädie</title>",
					"Successful. Recieved the correct Wikipedia response",
					"Error. Failed to recieve the correct Wikipedia response")
			)}
		)}
	})	
)}

	// setup SNI proxy for wildcard arbitrary domains

const (

	EgressWildcardEntry = `
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



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

package ossm

import (
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestExternalPrometheus(t *testing.T) {
	NewTest(t).Groups(Full).Run(func(t TestHelper) {
		t.Log("This test checks if Prometheus metrics are being honored")

		ns := "bookinfo"

		DeployControlPlane(t)

		t.LogStep("Install bookinfo and sleep pod")
		app.InstallAndWaitReady(t, app.Sleep(ns))
		t.Cleanup(func() {
			app.Uninstall(t, app.Sleep(ns))
		})

		retry.UntilSuccess(t, func(t TestHelper) {
			shell.Execute(t, "istioctl pc endpoint deploy/sleep",
				assert.OutputContains("sleep",
					"Sleep was discovered",
					"Expected sleep to be discovered, but it was not found"),
				assert.OutputContains("httpbin",
					"Httpbin was discovered",
					"Expected Httpbin to be discovered, but it was not found."))
		})

		assertRequestFailure := func(url string) {
			t.LogStepf("Confirm that request to %s fails", url)
			execInSleepPod(t, ns,
				fmt.Sprintf(`curl -sSL -o /dev/null -w "%%{http_code}" %s || echo "connection failed"`, url),
				assert.OutputContains("connection failed",
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

		oc.Label(t, "", "namespace", ns, "istio-discovery=enabled")

		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  addons: 
    grafana: 
      enabled: false
    kiali: 
      enabled: false
    prometheus: 
  enabled: false
meshConfig: 
extensionProviders: 
- name: prometheus
  prometheus: {}
gateways: 
egress: 
  enabled: false
openshiftRoute: 
  enabled: false
security: 
dataPlane: 
  mtls: true
manageNetworkPolicy: false
tracing: 
type: None`)

		t.Cleanup(func() {
			oc.Patch(t, meshNamespace,
				"smcp", smcpName,
				"json",
				`[{"op": "remove", "path": "/spec/meshConfig"}]`)
		})

		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		retry.UntilSuccess(t, func(t TestHelper) {
			shell.Execute(t, "istioctl pc endpoint deploy/sleep",
				assert.OutputContains("sleep",
					"Sleep was discovered",
					"Expected sleep to be discovered, but it was not found"),
				assert.OutputDoesNotContain("httpbin",
					"Httpbin was not discovered",
					"Expected Httpbin to not be discovered, but it was."))
		})
	})
}




const (
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
)
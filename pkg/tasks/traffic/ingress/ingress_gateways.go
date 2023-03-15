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

package ingress

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/request"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestIngressGateways(t *testing.T) {
	NewTest(t).LegacyID("T8").Groups(Full, InterOp).Run(func(t TestHelper) {
		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		app.InstallAndWaitReady(t, app.Httpbin(ns))

		t.NewSubTest("TrafficManagement_ingress_status_200_test").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, httpbinGateway1)
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					fmt.Sprintf("http://%s/status/200", gatewayHTTP),
					request.WithHost("httpbin.example.com"),
					assert.ResponseStatus(http.StatusOK))
			})
		})

		t.NewSubTest("TrafficManagement_ingress_headers_test").Run(func(t TestHelper) {
			oc.ApplyString(t, ns, httpbinGateway2)
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					fmt.Sprintf("http://%s/headers", gatewayHTTP),
					nil,
					assert.ResponseStatus(http.StatusOK),
				)
			})
		})
	})
}

const httpbinGateway1 = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: httpbin-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "httpbin.example.com"
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: httpbin
spec:
  hosts:
  - "httpbin.example.com"
  gateways:
  - httpbin-gateway
  http:
  - match:
    - uri:
        prefix: /status
    - uri:
        prefix: /delay
    route:
    - destination:
        port:
          number: 8000
        host: httpbin`

const httpbinGateway2 = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: httpbin-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*"
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: httpbin
spec:
  hosts:
  - "*"
  gateways:
  - httpbin-gateway
  http:
  - match:
    - uri:
        prefix: /headers
    route:
    - destination:
        port:
          number: 8000
        host: httpbin`

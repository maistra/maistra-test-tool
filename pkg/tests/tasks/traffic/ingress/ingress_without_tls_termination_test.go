// Copyright 2023 Red Hat, Inc.
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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestIngressWithoutTlsTermination validates configuring a HTTPS ingress access to a HTTPS service
func TestIngressWithoutTlsTermination(t *testing.T) {
	test.NewTest(t).Id("T10").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		ns := "bookinfo"

		t.Cleanup(func() {
			app.Uninstall(t, app.Nginx(ns))
		})

		t.Log("This test validates configuring a HTTPS ingress access to a HTTPS service.")
		t.Log("Doc reference: https://istio.io/v1.14/docs/tasks/traffic-management/ingress/ingress-sni-passthrough/")

		t.LogStep("Install nginx")
		app.InstallAndWaitReady(t, app.Nginx(ns))
		checkNginx(t, ns, "127.0.0.1", "8443", "")

		t.NewSubTest("configure a passthrough ingress gateway").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, nginxIngressGateway)
			})
			t.LogStep("Configure a passthrough ingress gateway")
			oc.ApplyString(t, ns, nginxIngressGateway)
			gatewayHTTP := istio.GetIngressGatewayHost(t, meshNamespace)
			secureIngressPort := istio.GetIngressGatewaySecurePort(t, meshNamespace)
			checkNginx(t, ns, gatewayHTTP, secureIngressPort, nginxServerCACert)
		})

	})
}

func checkNginx(t test.TestHelper, ns string, ingressHost string, port string, caCert string) {
	t.Log("Verify nginx server is running.")
	certFlag := "-k"
	if caCert != "" {
		certFlag = "--cacert " + caCert
	}

	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("run=my-nginx", ns),
			"istio-proxy",
			fmt.Sprintf(`curl -sS -v %s --resolve nginx.example.com:%s:%s https://nginx.example.com:%s`, certFlag, port, ingressHost, port),
			assert.OutputContains(
				"Welcome to nginx",
				"Got expected Welcome to nginx message",
				"Expected return message Welcome to nginx, but failed"))
	})
}

const nginxIngressGateway = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: mygateway
spec:
  selector:
    istio: ingressgateway # use istio default ingress gateway
  servers:
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: PASSTHROUGH
    hosts:
    - nginx.example.com
---
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: nginx
spec:
  hosts:
  - nginx.example.com
  gateways:
  - mygateway
  tls:
  - match:
    - port: 443
      sniHosts:
      - nginx.example.com
    route:
    - destination:
        host: my-nginx
        port:
          number: 443
`

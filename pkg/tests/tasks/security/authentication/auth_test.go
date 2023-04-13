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

package authentication

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/request"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAuthPolicy(t *testing.T) {
	NewTest(t).Id("T18").Groups(Full, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)

		meshNamespace := env.GetDefaultMeshNamespace()

		t.Cleanup(func() {
			oc.RecreateNamespace(t, "foo", "bar", "legacy")
		})

		app.InstallAndWaitReady(t,
			app.Httpbin("foo"),
			app.Httpbin("bar"),
			app.HttpbinNoSidecar("legacy"),
			app.Sleep("foo"),
			app.Sleep("bar"),
			app.SleepNoSidecar("legacy"))

		fromNamespaces := []string{"foo", "bar", "legacy"}
		toNamespaces := []string{"foo", "bar"}

		t.LogStep("Check connectivity from namespaces foo, bar, and legacy to namespaces foo and bar")
		retry.UntilSuccess(t, func(t TestHelper) {
			for _, from := range fromNamespaces {
				for _, to := range toNamespaces {
					assertConnectionSucessful(t, from, to)
				}
			}
		})

		t.NewSubTest("enable auto mTLS").Run(func(t TestHelper) {
			t.LogStep("Check if mTLS is enabled in foo")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", "foo"),
					"sleep",
					"curl http://httpbin.foo:8000/headers -s",
					assert.OutputContains("X-Forwarded-Client-Cert",
						"mTLS is enabled in namespace foo (X-Forwarded-Client-Cert header is present)",
						"mTLS is not enabled in namespace foo (X-Forwarded-Client-Cert header is not present)"))
			})

			t.LogStep("Check that mTLS is NOT enabled in legacy")
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.Exec(t,
					pod.MatchingSelector("app=sleep", "foo"),
					"sleep",
					"curl http://httpbin.legacy:8000/headers -s",
					assert.OutputDoesNotContain("X-Forwarded-Client-Cert",
						"mTLS is not enabled in namespace legacy (X-Forwarded-Client-Cert header is not present)",
						"mTLS is enabled in namespace legacy, but shouldn't be (X-Forwarded-Client-Cert header is present when it shouldn't be)"))
			})
		})

		t.NewSubTest("enable global mTLS STRICT mode").Run(func(t TestHelper) {
			t.LogStep("Enable mTLS STRICT mode globally")
			oc.ApplyString(t, meshNamespace, PeerAuthenticationMTLSStrict)
			t.Cleanup(func() {
				oc.DeleteFromString(t, meshNamespace, PeerAuthenticationMTLSStrict)
			})

			retry.UntilSuccess(t, func(t TestHelper) {
				from := "legacy"
				for _, to := range []string{"foo", "bar"} {
					oc.Exec(t,
						pod.MatchingSelector("app=sleep", from),
						"sleep",
						fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}" || echo "failed to connect"`, to, from, to),
						assert.OutputDoesNotContain("200",
							"Global mTLS expected 000",
							"Response 000 as expected"))
				}
			})
		})

		t.NewSubTest("namespace policy mtls").Run(func(t TestHelper) {
			t.LogStep("Enable mutual TLS per namespace")
			oc.ApplyString(t, "foo", PeerAuthenticationMTLSStrict)
			t.Cleanup(func() {
				oc.DeleteFromString(t, "foo", PeerAuthenticationMTLSStrict)
			})
			retry.UntilSuccess(t, func(t TestHelper) {
				for _, from := range []string{"foo", "bar", "legacy"} {
					for _, to := range []string{"foo", "bar"} {
						if from == "legacy" && to == "foo" {
							assertConnectionFailure(t, from, to)
						} else {
							assertConnectionSucessful(t, from, to)
						}
					}
				}
			})
		})

		t.NewSubTest("workload policy mtls").Run(func(t TestHelper) {
			t.LogStep("Enable mutual TLS per workload")
			oc.ApplyString(t, "bar", WorkloadPolicyStrict)
			t.Cleanup(func() {
				oc.DeleteFromString(t, "bar", WorkloadPolicyStrict)
			})
			retry.UntilSuccess(t, func(t TestHelper) {
				assertConnectionFailure(t, "legacy", "bar")
			})

			t.LogStep("Refine mutual TLS per port")
			oc.ApplyString(t, "bar", PortPolicy)
			retry.UntilSuccess(t, func(t TestHelper) {
				assertConnectionSucessful(t, "legacy", "bar")
			})
		})

		t.NewSubTest("policy precedence mtls").Run(func(t TestHelper) {
			t.LogStep("Overwrite foo namespace policy by a workload policy")
			oc.ApplyString(t, "foo", OverwritePolicy)
			t.Cleanup(func() {
				oc.DeleteFromString(t, "foo", OverwritePolicy)
			})
			retry.UntilSuccess(t, func(t TestHelper) {
				assertConnectionSucessful(t, "legacy", "foo")
			})
		})

		ingressGatewayHost := istio.GetIngressGatewayHost(t, meshNamespace)
		headersURL := fmt.Sprintf("http://%s/headers", ingressGatewayHost)

		t.NewSubTest("end-user JWT").Run(func(t TestHelper) {
			t.Log("End-user authentication")

			t.LogStep("Apply httpbin gateway")
			oc.ApplyString(t, "foo", HttpbinGateway)

			t.LogStep("Check httpbin request is successful")
			retry.UntilSuccess(t, func(t TestHelper) {
				assertResponseStatus(t, headersURL, nil, http.StatusOK)
			})

			t.LogStep("Apply a JWT policy")
			oc.ApplyTemplate(t, meshNamespace, JWTAuthPolicyTemplate, ossm.Smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, meshNamespace, JWTAuthPolicyTemplate, ossm.Smcp)
			})

			t.LogStep("Check whether request without token returns 200")
			retry.UntilSuccess(t, func(t TestHelper) {
				assertResponseStatus(t, headersURL, nil, http.StatusOK)
			})

			t.LogStep("Check whether request with an invalid token returns 401")
			retry.UntilSuccess(t, func(t TestHelper) {
				assertResponseStatus(t, headersURL, request.WithHeader("Authorization", "Bearer deadbeef"), http.StatusUnauthorized)
			})

			t.LogStep("Check whether request with a valid token returns 200")
			token := string(curl.Request(t, "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt", nil))
			token = strings.Trim(token, "\n")
			retry.UntilSuccess(t, func(t TestHelper) {
				assertResponseStatus(t, headersURL, request.WithHeader("Authorization", "Bearer "+token), http.StatusOK)
			})

			// skip gen-jwt.py and test JWT expires
		})

		t.NewSubTest("end-user require JWT").Run(func(t TestHelper) {
			t.Log("Require a valid token")
			oc.ApplyTemplate(t, meshNamespace, RequireTokenPolicyTemplate, ossm.Smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, meshNamespace, RequireTokenPolicyTemplate, ossm.Smcp)
			})

			retry.UntilSuccess(t, func(t TestHelper) {
				assertResponseStatus(t, headersURL, nil, http.StatusForbidden)
			})
		})

		t.NewSubTest("end-user require JWT per path").Run(func(t TestHelper) {
			t.Log("Require valid tokens per-path")
			oc.ApplyTemplate(t, meshNamespace, RequireTokenPathPolicyTemplate, ossm.Smcp)
			t.Cleanup(func() {
				oc.DeleteFromTemplate(t, meshNamespace, RequireTokenPathPolicyTemplate, ossm.Smcp)
			})

			retry.UntilSuccess(t, func(t TestHelper) {
				assertResponseStatus(t, headersURL, nil, http.StatusForbidden)

				ipURL := fmt.Sprintf("http://%s/ip", ingressGatewayHost)
				assertResponseStatus(t, ipURL, nil, http.StatusOK)
			})
		})
	})
}

func assertResponseStatus(t TestHelper, url string, requestOption curl.RequestOption, statusCode int) {
	curl.Request(t, url, requestOption, assert.ResponseStatus(statusCode))
}

const (
	WorkloadPolicyStrict = `
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: httpbin
  namespace: bar
spec:
  selector:
    matchLabels:
      app: httpbin
  mtls:
    mode: STRICT
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: httpbin
spec:
  host: httpbin.bar.svc.cluster.local
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
`

	PortPolicy = `
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: httpbin
  namespace: bar
spec:
  selector:
    matchLabels:
      app: httpbin
  mtls:
    mode: STRICT
  portLevelMtls:
    8000:
      mode: DISABLE
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: httpbin
spec:
  host: httpbin.bar.svc.cluster.local
  trafficPolicy:
    tls:
      mode: ISTIO_MUTUAL
    portLevelSettings:
    - port:
        number: 8000
      tls:
        mode: DISABLE
`

	OverwritePolicy = `
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: overwrite-example
  namespace: foo
spec:
  selector:
    matchLabels:
      app: httpbin
  mtls:
    mode: DISABLE
---
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: overwrite-example
spec:
  host: httpbin.foo.svc.cluster.local
  trafficPolicy:
    tls:
      mode: DISABLE
`

	HttpbinGateway = `
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: httpbin-gateway
  namespace: foo
spec:
  selector:
    istio: ingressgateway # use Istio default gateway implementation
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
  namespace: foo
spec:
  hosts:
  - "*"
  gateways:
  - httpbin-gateway
  http:
  - route:
    - destination:
        port:
          number: 8000
        host: httpbin.foo.svc.cluster.local
`

	JWTAuthPolicyTemplate = `
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-example
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  jwtRules:
  - issuer: testing@secure.istio.io
    jwksUri: https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/jwks.json
`

	RequireTokenPolicyTemplate = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: frontend-ingress
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  action: DENY
  rules:
  - from:
    - source:
        notRequestPrincipals: ["*"]
`

	RequireTokenPathPolicyTemplate = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: frontend-ingress
spec:
  selector:
    matchLabels:
      istio: ingressgateway
  action: DENY
  rules:
  - from:
    - source:
        notRequestPrincipals: ["*"]
    to:
    - operation:
        paths: ["/headers"]
`
)

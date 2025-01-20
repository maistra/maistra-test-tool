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

package authorization

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAuthorizationJWT(t *testing.T) {
	test.NewTest(t).Id("T22").Groups(test.Full, test.InterOp, test.ARM, test.Persistent).Run(func(t test.TestHelper) {
		ns := "foo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		t.Log("This test validates authorization policies with JWT Token")

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Sleep(ns))

		t.LogStep("Check if httpbin returns 200 OK when no authorization policies are in place")
		app.AssertSleepPodRequestSuccess(t, ns, "http://httpbin:8000/ip")

		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt"
		token := string(curl.Request(t, jwtURL, nil))

		groupURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/groups-scope.jwt"
		tokenGroup := string(curl.Request(t, groupURL, nil))

		headersUrl := "http://httpbin:8000/headers"

		t.Cleanup(func() {
			oc.DeleteFromString(t, ns, JWTExampleRule)
		})
		oc.ApplyString(t, ns, JWTExampleRule)

		t.NewSubTest("Allow requests with valid JWT and list-typed claims").Run(func(t test.TestHelper) {
			t.LogStep("Verify that a request with an invalid JWT is denied")
			app.AssertSleepPodRequestUnauthorized(t, ns, headersUrl, app.CurlOpts{Headers: []string{bearerTokenHeader("invalidToken")}})

			t.LogStep("Verify that a request without a JWT is allowed because there is no authorization policy")
			app.AssertSleepPodRequestSuccess(t, ns, headersUrl)
		})

		t.NewSubTest("Security authorization allow JWT requestPrincipal").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, JWTRequireRule)
			})
			oc.ApplyString(t, ns, JWTRequireRule)
			t.LogStep("Verify that a request with a valid JWT is allowed")
			app.AssertSleepPodRequestSuccess(t, ns, headersUrl, app.CurlOpts{Headers: []string{bearerTokenHeader(token)}})

			t.LogStep("Verify request without a JWT is denied")
			app.AssertSleepPodRequestForbidden(t, ns, headersUrl)
		})

		t.NewSubTest("Security authorization allow JWT claims group").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, JWTGroupClaimRule)
			})
			oc.ApplyString(t, ns, JWTGroupClaimRule)
			t.LogStep("Verify that a request with the JWT that includes group1 in the groups claim is allowed")
			app.AssertSleepPodRequestSuccess(t, ns, headersUrl, app.CurlOpts{Headers: []string{bearerTokenHeader(tokenGroup)}})

			t.LogStep("Verify that a request with a JWT, which does not have the groups claim is rejected")
			app.AssertSleepPodRequestForbidden(t, ns, headersUrl, app.CurlOpts{Headers: []string{bearerTokenHeader(token)}})
		})
	})
}

func bearerTokenHeader(token string) string {
	return fmt.Sprintf("Authorization: Bearer %s", token)
}

const (
	JWTExampleRule = `
apiVersion: security.istio.io/v1beta1
kind: RequestAuthentication
metadata:
  name: jwt-example
spec:
  selector:
    matchLabels:
      app: httpbin
  jwtRules:
  - issuer: testing@secure.istio.io
    jwksUri: https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/jwks.json
`
	JWTRequireRule = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: require-jwt
spec:
  selector:
    matchLabels:
      app: httpbin
  action: ALLOW
  rules:
  - from:
    - source:
       requestPrincipals: ["testing@secure.istio.io/testing@secure.istio.io"]
`

	JWTGroupClaimRule = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: require-jwt
spec:
  selector:
    matchLabels:
      app: httpbin
  action: ALLOW
  rules:
  - from:
    - source:
       requestPrincipals: ["testing@secure.istio.io/testing@secure.istio.io"]
    when:
    - key: request.auth.claims[groups]
      values: ["group1"]
`
)

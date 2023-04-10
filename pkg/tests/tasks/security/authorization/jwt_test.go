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
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAuthorizationJWT(t *testing.T) {
	test.NewTest(t).Id("T22").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "foo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
		})

		t.Log("This test validates authorization policies with JWT Token")

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Sleep(ns))

		t.LogStep("Check if httpbin returns 200 OK when no authorization policies are in place")
		retry.UntilSuccess(t, func(t test.TestHelper) {
			oc.Exec(t,
				pod.MatchingSelector("app=sleep", ns),
				"sleep",
				httpbinRequest("GET", "/ip"),
				assert.OutputContains(
					"200",
					"Got expected 200 OK from httpbin",
					"Expected 200 OK from httpbin, but got a different HTTP code"))
		})

		jwtURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/demo.jwt"
		token, err := util.Shell(`curl %s -s`, jwtURL)
		token = strings.Trim(token, "\n")
		if err != nil {
			t.Error("message")
		}

		groupURL := "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/groups-scope.jwt"
		tokenGroup, err := util.ShellMuteOutput(`curl %s -s`, groupURL)
		tokenGroup = strings.Trim(tokenGroup, "\n")
		if err != nil {
			t.Error("message")
		}

		t.NewSubTest("Allow requests with valid JWT and list-typed claims").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, JWTExampleRule)
			})
			oc.ApplyString(t, ns, JWTExampleRule)

			t.LogStep("Verify that a request with an invalid JWT is denied")
			assertRequestJWTDenied_401(t, ns, curl_JWT("Authorization: Bearer invalidToken"))

			t.LogStep("Verify that a request without a JWT is allowed because there is no authorization policy")
			assertRequestJWTAccepted(t, ns, curl_JWT())

		})

		t.NewSubTest("Security authorization allow JWT requestPrincipal").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, JWTRequireRule)
			})
			oc.ApplyString(t, ns, JWTRequireRule)

			t.LogStep("Verify that a request with a valid JWT is allowed")
			assertRequestJWTAccepted(t, ns, curl_JWT("Authorization: Bearer %s", token))

			t.LogStep("Verify request without a JWT is denied")
			assertRequestJWTDenied(t, ns, curl_JWT())

		})

		t.NewSubTest("Security authorization allow JWT claims group").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, JWTGroupClaimRule)
			})
			oc.ApplyString(t, ns, JWTGroupClaimRule)

			t.LogStep("Verify that a request with the JWT that includes group1 in the groups claim is allowed")
			assertRequestJWTAccepted(t, ns, curl_JWT("Authorization: Bearer %s", tokenGroup))

			t.LogStep("Verify that a request with a JWT, which does not have the groups claim is rejected")
			assertRequestJWTDenied(t, ns, curl_JWT("Authorization: Bearer %s", token))
		})
	})
}

func curl_JWT(headers ...string) string {
	headerArgs := ""
	for _, header := range headers {
		headerArgs += fmt.Sprintf(` -H "%s"`, header)
	}
	return fmt.Sprintf(`curl "http://httpbin:8000/headers" -sS -o /dev/null %s -w "%%%%{http_code}\n"`, headerArgs)
}

func assertRequestJWTDenied(t test.TestHelper, ns string, curlCommand string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			curlCommand,
			assert.OutputContains(
				"403",
				"Got the expected 403 Forbidden response",
				"Expected the JWT Authorization Policy to reject request (expected HTTP status 403), but got a different HTTP code"))
	})
}

func assertRequestJWTDenied_401(t test.TestHelper, ns string, curlCommand string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			curlCommand,
			assert.OutputContains(
				"401",
				"Got the expected 401 Unauthorized response",
				"Expected the JWT AuthorizationPolicy to reject request (expected HTTP status 401), but got a different HTTP code"))
	})
}

func assertRequestJWTAccepted(t test.TestHelper, ns string, curlCommand string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		oc.Exec(t,
			pod.MatchingSelector("app=sleep", ns),
			"sleep",
			curlCommand,
			assert.OutputContains(
				"200",
				"Got the expected 200 OK response for the JWT request",
				"Expected the JWT AuthorizationPolicy to accept request (expected HTTP status 200), but got a different HTTP code"))
	})
}

const (
	JWTExampleRule = `
apiVersion: "security.istio.io/v1beta1"
kind: "RequestAuthentication"
metadata:
  name: "jwt-example"
  namespace: foo
spec:
  selector:
    matchLabels:
      app: httpbin
  jwtRules:
  - issuer: "testing@secure.istio.io"
    jwksUri: "https://raw.githubusercontent.com/istio/istio/release-1.9/security/tools/jwt/samples/jwks.json"
`
	JWTRequireRule = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: require-jwt
  namespace: foo
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
  namespace: foo
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

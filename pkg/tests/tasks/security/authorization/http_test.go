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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestAuthorizationHTTPTraffic validates authorization policies for HTTP traffic.
func TestAuthorizationHTTPTraffic(t *testing.T) {
	NewTest(t).Id("T20").Groups(Full, ARM, InterOp, Disconnected).Run(func(t TestHelper) {
		ns := "bookinfo"
		t.Cleanup(func() {
			oc.Patch(t,
				meshNamespace, "smcp", smcpName, "merge",
				`{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}`,
			)
			oc.RecreateNamespace(t, ns)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.Log("This test validates authorization policies for HTTP traffic.")
		t.Log("Doc reference: https://istio.io/v1.14/docs/tasks/security/authorization/authz-http/")

		ossm.DeployControlPlane(t)

		t.LogStep("Enable Service Mesh Control Plane mTLS")
		oc.Patch(t,
			meshNamespace, "smcp", smcpName, "merge",
			`{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}`,
		)

		t.LogStep("Install bookinfo with mTLS")
		app.InstallAndWaitReady(t, app.BookinfoWithMTLS(ns))
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		productPageURL := app.BookinfoProductPageURL(t, meshNamespace)

		t.NewSubTest("deny all http traffic to bookinfo").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, DenyAllPolicy)
			})
			t.LogStep("Apply policy that denies all HTTP requests to bookinfo workloads")
			oc.ApplyString(t, ns, DenyAllPolicy)

			t.LogStep("Verify that GET request is denied")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					productPageURL,
					nil,
					assert.ResponseContains("RBAC: access denied"),
				)
			})
		})

		t.NewSubTest("only allow HTTP GET request to the productpage workload").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ProductpageGETPolicy)
				oc.DeleteFromString(t, ns, DenyAllPolicy)
			})
			t.LogStep("Apply policies that allow access with GET method to the productpage workload and deny requests to other workloads")
			oc.ApplyString(t, ns, DenyAllPolicy)
			oc.ApplyString(t, ns, ProductpageGETPolicy)

			t.LogStep("Verify that GET request to the productpage is allowed and fetching other services is denied")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					productPageURL,
					nil,
					assert.ResponseContains("Error fetching product details"),
					assert.ResponseContains("Error fetching product reviews"),
				)
			})
		})

		t.NewSubTest("allow HTTP GET requests to all bookinfo workloads").Run(func(t TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, RatingsGETPolicy)
				oc.DeleteFromString(t, ns, ReviewsGETPolicy)
				oc.DeleteFromString(t, ns, DetailsGETPolicy)
				oc.DeleteFromString(t, ns, ProductpageGETPolicy)
				oc.DeleteFromString(t, ns, DenyAllPolicy)
			})
			t.LogStep("Apply policies that allow HTTP GET requests to all bookinfo workloads")
			oc.ApplyString(t, ns, DenyAllPolicy)
			oc.ApplyString(t, ns, ProductpageGETPolicy)
			oc.ApplyString(t, ns, DetailsGETPolicy)
			oc.ApplyString(t, ns, ReviewsGETPolicy)
			oc.ApplyString(t, ns, RatingsGETPolicy)

			t.LogStep("Verify that GET requests are allowed to all bookinfo workloads")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					productPageURL,
					nil,
					assert.ResponseDoesNotContain("RBAC: access denied"),
					assert.ResponseDoesNotContain("Error fetching product details"),
					assert.ResponseDoesNotContain("Error fetching product reviews"),
					assert.ResponseDoesNotContain("Ratings service currently unavailable"),
				)
			})
		})
	})
}

const (
	DenyAllPolicy = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: allow-nothing
  namespace: bookinfo
spec:
  {}
`

	ProductpageGETPolicy = `
apiVersion: "security.istio.io/v1beta1"
kind: "AuthorizationPolicy"
metadata:
  name: "productpage-viewer"
  namespace: bookinfo
spec:
  selector:
    matchLabels:
      app: productpage
  action: ALLOW
  rules:
  - to:
    - operation:
        methods: ["GET"]
`

	DetailsGETPolicy = `
apiVersion: "security.istio.io/v1beta1"
kind: "AuthorizationPolicy"
metadata:
  name: "details-viewer"
  namespace: bookinfo
spec:
  selector:
    matchLabels:
      app: details
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/bookinfo/sa/bookinfo-productpage"]
    to:
    - operation:
        methods: ["GET"]
`

	ReviewsGETPolicy = `
apiVersion: "security.istio.io/v1beta1"
kind: "AuthorizationPolicy"
metadata:
  name: "reviews-viewer"
  namespace: bookinfo
spec:
  selector:
    matchLabels:
      app: reviews
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/bookinfo/sa/bookinfo-productpage"]
    to:
    - operation:
        methods: ["GET"]
`

	RatingsGETPolicy = `
apiVersion: "security.istio.io/v1beta1"
kind: "AuthorizationPolicy"
metadata:
  name: "ratings-viewer"
  namespace: bookinfo
spec:
  selector:
    matchLabels:
      app: ratings
  action: ALLOW
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/bookinfo/sa/bookinfo-reviews"]
    to:
    - operation:
        methods: ["GET"]
`
)

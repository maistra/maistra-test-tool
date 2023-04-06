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
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestAuthoriztionHTTPTraffic(t *testing.T) {
	test.NewTest(t).Id("T20").Groups(test.Full, test.ARM, test.InterOp).Run(func(t test.TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "bookinfo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
			shell.Execute(t,
				fmt.Sprintf(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`,
					meshNamespace, smcpName))
			shell.Execute(t,
				fmt.Sprintf(`kubectl -n %s wait --for condition=Ready smcp/%s --timeout 180s`,
					meshNamespace, smcpName))
		})

		t.Log("This test validates authorization policies for HTTP traffic.")
		t.Log("Doc reference: https://istio.io/v1.14/docs/tasks/security/authorization/authz-http/")

		t.LogStep("Enable Service Mesh Control Plane mTLS")
		shell.Execute(t,
			fmt.Sprintf(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`,
				meshNamespace, smcpName))
		shell.Execute(t,
			fmt.Sprintf(`kubectl -n %s wait --for condition=Ready smcp/%s --timeout 180s`,
				meshNamespace, smcpName))

		t.LogStep("Install bookinfo with mTLS")
		app.InstallAndWaitReady(t, app.BookinfoWithMTLS(ns))
		GATEWAY_URL := shell.Execute(t,
			fmt.Sprintf(`kubectl get routes -n %s istio-ingressgateway -o jsonpath='{.spec.host}'`, meshNamespace))

		t.NewSubTest("deny all http traffic to bookinfo").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, DenyAllPolicy)
			})
			t.LogStep("Apply policy that denies all HTTP requests to bookinfo workloads")
			oc.ApplyString(t, ns, DenyAllPolicy)

			t.LogStep("Verify that GET request is denied")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputContains(
						"RBAC: access denied",
						"ProductPage returns expected message: 'RBAC: access denied'",
						"ProductPage didn't return 'RBAC: access denied'"),
				)
			})
		})

		t.NewSubTest("only allow HTTP GET request to the productpage workload").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.DeleteFromString(t, ns, ProductpageGETPolicy)
				oc.DeleteFromString(t, ns, DenyAllPolicy)
			})
			t.LogStep("Apply policies that allow access with GET method to the productpage workload and deny requests to other workloads")
			oc.ApplyString(t, ns, DenyAllPolicy)
			oc.ApplyString(t, ns, ProductpageGETPolicy)

			t.LogStep("Verify that GET request to the productpage is allowed and fetching other services is denied")
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputContains(
						"Error fetching product details",
						"ProductPage returns expected message: 'Error fetching product details'",
						"ProductPage didn't return 'Error fetching product details'"),
				)
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputContains(
						"Error fetching product reviews",
						"ProductPage returns expected message: 'Error fetching product reviews'",
						"ProductPage didn't return 'Error fetching product reviews'"),
				)
			})
		})

		t.NewSubTest("allow HTTP GET requests to all bookinfo workloads").Run(func(t test.TestHelper) {
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
			retry.UntilSuccess(t, func(t test.TestHelper) {
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputDoesNotContain(
						"RBAC: access denied",
						"ProductPage returns successfully",
						"ProductPage returns unexpected message: 'RBAC: access denied'"),
				)
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputDoesNotContain(
						"Error fetching product details",
						"ProductPage details returns successfully",
						"ProductPage returns unexpected message: 'Error fetching product details'"),
				)
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputDoesNotContain(
						"Error fetching product reviews",
						"ProductPage reviews returns successfully",
						"ProductPage returns unexpected message: 'Error fetching product reviews'"),
				)
				shell.Execute(t,
					fmt.Sprintf(`curl http://%s/productpage`, GATEWAY_URL),
					assert.OutputDoesNotContain(
						"Ratings service currently unavailable",
						"ProductPage ratings returns successfully",
						"ProductPage returns unexpected message: 'Ratings service currently unavailable'"),
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

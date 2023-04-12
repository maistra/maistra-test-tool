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
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

// External Authorization support the 2.4 SM. In the 2.3 SM, it is in techPreview. Please check the version and configure it.

func TestExtAuthz(t *testing.T) {
	test.NewTest(t).Id("T37").Groups(test.Full, test.InterOp).Run(func(t test.TestHelper) {
		hack.DisableLogrusForThisTest(t)

		ns := "foo"
		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns)
			oc.Patch(t, meshNamespace,
				"smcp", smcpName,
				"json",
				`[{"op": "remove", "path": "/spec/techPreview"}]`)
		})

		t.Log("This test validates authorization policies with a JWT Token")

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns), app.Sleep(ns))

		t.LogStep("Check if httpbin returns 200 OK when no authorization policies are in place")
		assertHttpbinRequestSucceeds(t, ns, httpbinRequest("GET", "/ip"))

		t.LogStep("Deploy the External Authorizer and Verify the sample external authorizer is up and running")
		oc.ApplyString(t, ns, ExternalAuthzService)

		oc.WaitDeploymentRolloutComplete(t, ns, "ext-authz")

		t.LogStep("Set envoyExtAuthzHttp extension provider in SMCP")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge",
			`{"spec": {"techPreview": {"meshConfig": {"extensionProviders": [{"envoyExtAuthzHttp": {"includeRequestHeadersInCheck": ["x-ext-authz"],"port": "8000","service": "ext-authz.foo.svc.cluster.local"},"name": "sample-ext-authz-http"}]}}}}`,
		)

		t.LogStep("Deploy the external authorization in the Authorization policy")
		t.Cleanup(func() {
			oc.DeleteFromString(t, ns, ExternalRoute)
		})
		oc.ApplyString(t, ns, ExternalRoute)

		t.LogStep("Verify a request to path /headers with header x-ext-authz: deny is denied by the sample ext_authz server:")
		assertRequestDenied(t, ns, httpbinRequest("GET", "/headers", "x-ext-authz: deny"))

		t.LogStep("Verify a request to path /headers with header x-ext-authz: allow is allowed by the sample ext_authz server")
		assertRequestAccepted(t, ns, httpbinRequest("GET", "/headers", "x-ext-authz: allow"))

		t.LogStep("Verify a request to path /ip is allowed and does not trigger the external authorization")
		assertHttpbinRequestSucceeds(t, ns, httpbinRequest("GET", "/ip"))

	})
}

const (
	ExternalRoute = `
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: ext-authz
  namespace: foo
spec:
  selector:
    matchLabels:
      app: httpbin
  action: CUSTOM
  provider:
    # The provider name must match the extension provider defined in the mesh config.
    # You can also replace this with sample-ext-authz-http to test the other external authorizer definition.
    name: sample-ext-authz-http
  rules:
  # The rules specify when to trigger the external authorizer.
  - to:
    - operation:
       paths: ["/headers"]
`

	ExternalAuthzService = `
apiVersion: v1
kind: Service
metadata:
  name: ext-authz
  labels:
    app: ext-authz
spec:
  ports:
  - name: http
    port: 8000
    targetPort: 8000
  - name: grpc
    port: 9000
    targetPort: 9000
  selector:
    app: ext-authz
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ext-authz
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ext-authz
  template:
    metadata:
      labels:
        app: ext-authz
    spec:
      containers:
      - image: gcr.io/istio-testing/ext-authz:latest
      readinessProbe:
        imagePullPolicy: IfNotPresent
        name: ext-authz
        ports:
        - containerPort: 8000
        - containerPort: 9000
`
)

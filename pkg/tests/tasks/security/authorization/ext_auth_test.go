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
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/version"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestEnvoyExtAuthzHttpExtensionProvider(t *testing.T) {
	NewTest(t).Id("T37").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.Skip("extensionProviders.envoyExtAuthzHttp was added in v2.3")
		}
		t.Log("This test validates authorization policies with a JWT Token")

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo), app.Sleep(ns.Foo))
		t.Cleanup(func() {
			app.Uninstall(t, app.Httpbin(ns.Foo), app.Sleep(ns.Foo))
		})

		t.LogStep("Check if httpbin returns 200 OK when no authorization policies are in place")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/ip")

		t.LogStep("Deploy the External Authorizer and Verify the sample external authorizer is up and running")
		oc.ApplyTemplate(t, ns.Foo, ExternalAuthzService, nil)
		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, ns.Foo, ExternalAuthzService, nil)
		})

		oc.WaitDeploymentRolloutComplete(t, ns.Foo, "ext-authz")

		t.LogStep("Set envoyExtAuthzHttp extension provider in SMCP")
		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  techPreview:
    meshConfig:
      extensionProviders:
        - name: sample-ext-authz-http
          envoyExtAuthzHttp:
            headersToDownstreamOnDeny:
              - set-cookie
              - location
            headersToDownstreamOnAllow:
              - set-cookie
              - location
            headersToUpstreamOnAllow:
              - location
              - email
              - authorization
              - path
              - x-auth-request-user
              - x-auth-request-email
              - x-auth-request-access-token
            includeRequestHeadersInCheck:
              - x-ext-authz
            port: "8000"
            service: ext-authz.foo.svc.cluster.local`)

			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json",
					`[{"op": "remove", "path": "/spec/techPreview"}]`)
			})

		} else {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  meshConfig:
    extensionProviders:
      - name: sample-ext-authz-http
        envoyExtAuthzHttp:
          headersToDownstreamOnDeny:
            - set-cookie
            - location
          headersToDownstreamOnAllow:
            - set-cookie
            - location
          headersToUpstreamOnAllow:
            - location
            - email
            - authorization
            - path
            - x-auth-request-user
            - x-auth-request-email
            - x-auth-request-access-token
          includeRequestHeadersInCheck:
            - x-ext-authz
          port: 8000
          service: ext-authz.foo.svc.cluster.local`)

			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json",
					`[{"op": "remove", "path": "/spec/meshConfig"}]`)
			})
		}

		t.LogStep("Deploy the external authorization in the Authorization policy")
		t.Cleanup(func() {
			oc.DeleteFromString(t, ns.Foo, ExternalRoute)
		})
		oc.ApplyString(t, ns.Foo, ExternalRoute)

		t.LogStep("Verify a request to path /headers with header x-ext-authz: deny is denied by the sample ext_authz server:")
		app.AssertSleepPodRequestForbidden(t, ns.Foo, "http://httpbin:8000/headers", app.CurlOpts{Headers: []string{"x-ext-authz: deny"}})

		t.LogStep("Verify a request to path /headers with header x-ext-authz: allow is allowed by the sample ext_authz server")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/headers", app.CurlOpts{Headers: []string{"x-ext-authz: allow"}})

		t.LogStep("Verify a request to path /ip is allowed and does not trigger the external authorization")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/ip")
	})
}

func TestEnvoyExtAuthzGrpcExtensionProvider(t *testing.T) {
	NewTest(t).Id("T42").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_3) {
			t.Skip("extensionProviders.envoyExtAuthzGrpc is not supported in versions below v2.3")
		}
		t.Log("This test validates authorization policies with a JWT Token")

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t, app.Httpbin(ns.Foo), app.Sleep(ns.Foo))
		t.Cleanup(func() {
			app.Uninstall(t, app.Httpbin(ns.Foo), app.Sleep(ns.Foo))
		})

		t.LogStep("Check if httpbin returns 200 OK when no authorization policies are in place")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/ip")

		t.LogStep("Deploy the External Authorizer and Verify the sample external authorizer is up and running")
		oc.ApplyTemplate(t, ns.Foo, ExternalAuthzService, nil)
		t.Cleanup(func() {
			oc.DeleteFromTemplate(t, ns.Foo, ExternalAuthzService, nil)
		})

		oc.WaitDeploymentRolloutComplete(t, ns.Foo, "ext-authz")

		t.LogStep("Set envoyExtAuthzgRPC extension provider in SMCP")
		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  techPreview:
    meshConfig:
      extensionProviders:
        - name: sample-ext-authz-grpc
          envoyExtAuthzGrpc:
            includeRequestHeadersInCheck:
              - x-ext-authz
            port: "9000"
            service: ext-authz.foo.svc.cluster.local`)

			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json",
					`[{"op": "remove", "path": "/spec/techPreview"}]`)
			})

		} else {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  meshConfig:
    extensionProviders:
      - name: sample-ext-authz-grpc
        envoyExtAuthzGrpc:
          includeRequestHeadersInCheck:
            - x-ext-authz
          port: 9000
          service: ext-authz.foo.svc.cluster.local`)

			t.Cleanup(func() {
				oc.Patch(t, meshNamespace, "smcp", smcpName, "json",
					`[{"op": "remove", "path": "/spec/meshConfig"}]`)
			})
		}

		t.LogStep("Deploy the external authorization in the Authorization policy")
		t.Cleanup(func() {
			oc.DeleteFromString(t, ns.Foo, ExternalRouteGrpc)
		})
		oc.ApplyString(t, ns.Foo, ExternalRouteGrpc)

		t.LogStep("Verify a request to path /headers with header x-ext-authz: deny is denied by the sample ext_authz server:")
		app.AssertSleepPodRequestForbidden(t, ns.Foo, "http://httpbin:8000/headers", app.CurlOpts{Headers: []string{"x-ext-authz: deny"}})

		t.LogStep("Verify a request to path /headers with header x-ext-authz: allow is allowed by the sample ext_authz server")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/headers", app.CurlOpts{Headers: []string{"x-ext-authz: allow"}})

		t.LogStep("Verify a request to path /ip is allowed and does not trigger the external authorization")
		app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/ip")
	})
}

func TestEnvoyExtAuthzRequestPayloadTooLarge(t *testing.T) {
	NewTest(t).Groups(Full, ARM).Run(func(t TestHelper) {
		t.Log("Verify that Istio proxy doesn't fail with 'Request payload too large' error")
		t.Log("Reference: https://issues.redhat.com/browse/OSSM-5850")

		t.Cleanup(func() {
			oc.RecreateNamespace(t, ns.Foo, meshNamespace)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin and sleep")
		app.InstallAndWaitReady(t,
			app.Httpbin(ns.Foo),
			app.Sleep(ns.Foo))

		t.LogStep("Create 2MB file in the sleep pod")
		app.ExecInSleepPod(t, ns.Foo, "fallocate -l 2M /tmp/fakefile_2MB")
		assertFileUploadSuccess(t)

		t.LogStep("Deploy the External Authorizer and Verify the sample external authorizer is up and running")
		oc.ApplyTemplate(t, ns.Foo, ExternalAuthzService, nil)
		oc.WaitDeploymentRolloutComplete(t, ns.Foo, "ext-authz")

		t.LogStep("Deploy the external authorization in the Authorization policy")
		oc.ApplyString(t, ns.Foo, ExternalRoute)

		t.LogStep("Patch SMCP to enable envoyExtAuthzHttp with allowPartialMessage true and maxRequestBytes 1KB")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", uploadEnvoyExtSpec("true", 1024))
		oc.WaitSMCPReady(t, meshNamespace, smcpName)
		assertFileUploadSuccess(t)

		t.LogStep("Patch SMCP to set allowPartialMessage to false and increase maxRequestBytes to 5MB")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", uploadEnvoyExtSpec("false", 5000000))
		oc.WaitSMCPReady(t, meshNamespace, smcpName)
		assertFileUploadSuccess(t)
	})
}

func assertFileUploadSuccess(t TestHelper) {
	t.LogStep("Send a POST request with a large file (2MB)")
	app.AssertSleepPodRequestSuccess(t, ns.Foo, "http://httpbin:8000/post", app.CurlOpts{
		Method:  "POST",
		Headers: []string{"x-ext-authz: allow", "Content-Type: multipart/form-data"},
		Options: []string{"-F file=@/tmp/fakefile_2MB"}})
}

func uploadEnvoyExtSpec(AllowPartialMessage string, MaxRequestBytes int) string {
	return fmt.Sprintf(`
spec:
  meshConfig:
    extensionProviders:
    - name: sample-ext-authz-http
      envoyExtAuthzHttp:
        service: ext-authz.foo.svc.cluster.local
        port: 8000
        includeRequestHeadersInCheck: ["x-ext-authz"]
        includeRequestBodyInCheck:
          allowPartialMessage: %s
          maxRequestBytes: %d
          packAsBytes: true`, AllowPartialMessage, MaxRequestBytes)
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
       paths: ["/headers", "/post"]
`
	ExternalRouteGrpc = `
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
    name: sample-ext-authz-grpc
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
      - image: {{ image "ext-authz" }}
        imagePullPolicy: IfNotPresent
        name: ext-authz
        ports:
        - containerPort: 8000
        - containerPort: 9000
`
)

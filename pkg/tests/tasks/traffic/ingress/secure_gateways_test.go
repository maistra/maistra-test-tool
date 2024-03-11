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
	_ "embed"
	"net/http"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/request"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var (
	//go:embed yaml/httpbin-tls-gateway-https.yaml
	httpbinTLSGatewayHTTPS string

	//go:embed yaml/gateway-multiple-hosts.yaml
	gatewayMultipleHosts string

	//go:embed yaml/gateway-httpbin-mtls.yaml
	gatewayHttpbinMTLSYaml string

	//go:embed yaml/hello-world.yaml
	helloWorldTemplate string
)

func TestSecureGateways(t *testing.T) {
	NewTest(t).Id("T9").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		ns := "bookinfo"

		t.Log("This test verifies secure gateways.")
		t.Log("Doc reference: https://istio.io/latest/docs/tasks/traffic-management/ingress/secure-ingress/")

		t.Cleanup(func() {
			oc.DeleteSecret(t, meshNamespace, "httpbin-credential")
			oc.DeleteSecret(t, meshNamespace, "helloworld-credential")
			oc.RecreateNamespace(t, ns)
		})

		ossm.DeployControlPlane(t)

		t.LogStep("Install httpbin")
		app.InstallAndWaitReady(t, app.Httpbin(ns))
		oc.ApplyTemplate(t, ns, helloWorldTemplate, nil)
		oc.WaitDeploymentRolloutComplete(t, ns, "helloworld-v1")

		t.LogStep("Create TLS secrets")
		oc.CreateTLSSecret(t, meshNamespace, "httpbin-credential", httpbinSampleServerCertKey, httpbinSampleServerCert)
		oc.CreateTLSSecret(t, meshNamespace, "helloworld-credential", helloworldServerCertKey, helloworldServerCert)

		gatewayHost := istio.GetIngressGatewayHost(t, meshNamespace)
		gatewayPort := istio.GetIngressGatewaySecurePort(t, meshNamespace)
		helloWorldURL := "https://helloworld-v1.example.com:" + gatewayPort + "/hello"
		teapotURL := "https://httpbin.example.com:" + gatewayPort + "/status/418"

		t.NewSubTest("tls_single_host").Run(func(t TestHelper) {
			t.LogStep("Configure a TLS ingress gateway for a single host")
			oc.ApplyString(t, ns, httpbinTLSGatewayHTTPS)

			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
				createRouteWithTLS(t, meshNamespace, "httpbin.example.com", "https", "istio-ingressgateway", "passthrough")
			}

			t.LogStep("check if httpbin responds with teapot")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort),
					assert.ResponseContains("-=[ teapot ]=-"),
				)
			})
		})

		t.NewSubTest("tls_multiple_hosts").Run(func(t TestHelper) {
			t.LogStep("configure Gateway with multiple TLS hosts")
			oc.ApplyString(t, ns, gatewayMultipleHosts)

			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
				createRouteWithTLS(t, meshNamespace, "helloworld-v1.example.com", "https", "istio-ingressgateway", "passthrough")
				createRouteWithTLS(t, meshNamespace, "httpbin.example.com", "https", "istio-ingressgateway", "passthrough")
			}

			t.LogStep("check if helloworld-v1 responds with 200 OK")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					helloWorldURL,
					request.WithTLS(httpbinSampleCACert, "helloworld-v1.example.com", gatewayHost, gatewayPort),
					assert.ResponseStatus(http.StatusOK))
			})

			t.LogStep("check if httpbin responds with teapot")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort),
					assert.ResponseContains("-=[ teapot ]=-"))
			})
		})

		t.NewSubTest("mutual_tls").Run(func(t TestHelper) {
			t.LogStep("configure Gateway with tls.mode=Mutual")
			oc.CreateGenericSecretFromFiles(t, meshNamespace, "httpbin-credential",
				"tls.key="+httpbinSampleServerCertKey,
				"tls.crt="+httpbinSampleServerCert,
				"ca.crt="+httpbinSampleCACert)
			oc.ApplyString(t, ns, gatewayHttpbinMTLSYaml)

			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
				createRouteWithTLS(t, meshNamespace, "httpbin.example.com", "https", "istio-ingressgateway", "passthrough")
			}

			t.LogStep("check if SSL handshake fails when no client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort),
					assert.RequestFails(
						"request failed as expected",
						"expected request to fail because no client certificate was provided"),
					assert.RequestFailsWithAnyErrorMessages(
						[]string{
							"Get \"https://httpbin.example.com:443/status/418\": remote error: tls: certificate require",
							"Get \"https://httpbin.example.com:443/status/418\": remote error: tls: handshake failure"}, //FIPS OCP
						"request failed with expected error message",
						"request failed but with different error message"))
			})

			t.LogStep("check if SSL handshake succeeds when client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.
						WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort).
						WithClientCertificate(httpbinSampleClientCert, httpbinSampleClientCertKey),
					assert.ResponseContains("-=[ teapot ]=-"))
			})
		})

		t.NewSubTest("mutual_tls_with_crl").Run(func(t TestHelper) {
			t.Log("Reference: https://issues.redhat.com/browse/OSSM-414")
			if env.GetSMCPVersion().LessThan(version.SMCP_2_5) {
				t.Skip("Skipping until 2.5")
			}
			t.LogStep("configure Gateway with tls.mode=Mutual and provide CRL file")
			oc.CreateGenericSecretFromFiles(t, meshNamespace, "httpbin-credential",
				"tls.key="+httpbinSampleServerCertKey,
				"tls.crt="+httpbinSampleServerCert,
				"ca.crt="+httpbinSampleCACert,
				"ca.crl="+httpbinSampleCACrl)
			oc.ApplyString(t, ns, gatewayHttpbinMTLSYaml)

			createRouteWithTLS(t, meshNamespace, "httpbin.example.com", "https", "istio-ingressgateway", "passthrough")
			t.LogStep("check if SSL handshake fails when no client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort),
					assert.RequestFails(
						"request failed as expected",
						"expected request to fail because no client certificate was provided"),
					assert.RequestFailsWithAnyErrorMessages(
						// FIPS OCP has different message
						[]string{
							"Get \"https://httpbin.example.com:443/status/418\": remote error: tls: certificate require",
							"Get \"https://httpbin.example.com:443/status/418\": remote error: tls: handshake failure"}, //FIPS OCP
						"request failed with expected error message",
						"request failed but with different error message"))
			})

			t.LogStep("check if SSL handshake succeeds when client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.
						WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort).
						WithClientCertificate(httpbinSampleClientCert, httpbinSampleClientCertKey),
					assert.ResponseContains("-=[ teapot ]=-"))
			})

			t.LogStep("check if SSL handshake fails when revoked client certificate is given")
			retry.UntilSuccess(t, func(t TestHelper) {
				curl.Request(t,
					teapotURL,
					request.
						WithTLS(httpbinSampleCACert, "httpbin.example.com", gatewayHost, gatewayPort).
						WithClientCertificate(httpbinSampleClientRevokedCert, httpbinSampleClientRevokedCertKey),
					assert.RequestFails(
						"request failed as expected",
						"expected request to fail because revoked client certificate was provided"),
					assert.RequestFailsWithErrorMessage(
						"Get \"https://httpbin.example.com:443/status/418\": remote error: tls: revoked certificate",
						"request failed with expected error message",
						"request failed but with different error message"))
			})
		})
	})
}

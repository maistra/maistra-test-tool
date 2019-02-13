// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra

const (
	modelDir					= "testdata/modelDir"
	
	bookinfoYaml 				= "testdata/bookinfo/platform/kube/bookinfo.yaml"
	bookinfoGateway 			= "testdata/bookinfo/networking/bookinfo-gateway.yaml"
	bookinfoRuleAllYaml 		= "testdata/bookinfo/networking/destination-rule-all.yaml"
	bookinfoRuleAllTLSYaml 		= "testdata/bookinfo/networking/destination-rule-all-mtls.yaml"
	
	bookinfoAllv1Yaml			= "testdata/bookinfo/networking/virtual-service-all-v1.yaml"
	bookinfoRatingDelayYaml		= "testdata/bookinfo/networking/virtual-service-ratings-test-delay.yaml"
	bookinfoRatingDelayv2Yaml	= "testdata/bookinfo/networking/virtual-service-ratings-test-delay-2.yaml"
	bookinfoRatingAbortYaml		= "testdata/bookinfo/networking/virtual-service-ratings-test-abort.yaml"
	bookinfoRatingDBYaml 		= "testdata/bookinfo/networking/virtual-service-ratings-db.yaml"
	bookinfoRatingMySQLYaml		= "testdata/bookinfo/networking/virtual-service-ratings-mysql.yaml"
	bookinfoRatingMySQLv2Yaml 	= "testdata/bookinfo/networking/bookinfo-ratings-v2-mysql.yaml"
	bookinfoRatingMySQLServiceEntryYaml = "testdata/bookinfo/networking/bookinfo-ratings-mysql-service-entry.yaml"
	bookinfoReviewTestv2Yaml	= "testdata/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
	bookinfoReview50v3Yaml 		= "testdata/bookinfo/networking/virtual-service-reviews-50-v3.yaml"
	bookinfoReviewv3Yaml 		= "testdata/bookinfo/networking/virtual-service-reviews-v3.yaml"
	bookinfoReviewv2v3Yaml 		= "testdata/bookinfo/networking/virtual-service-reviews-jason-v2-v3.yaml"
	bookinfoReviewTimeoutYaml	= "testdata/bookinfo/networking/virtual-service-reviews-timeout.yaml"
	bookinfoDBYaml 				= "testdata/bookinfo/networking/bookinfo-db.yaml"
	bookinfoAddServiceAccountYaml = "testdata/bookinfo/platform/kube/bookinfo-add-serviceaccount.yaml"
	bookinfoRBACOnYaml 			= "testdata/bookinfo/platform/kube/rbac/rbac-config-ON.yaml"
	bookinfoNamespacePolicyYaml = "testdata/bookinfo/platform/kube/rbac/namespace-policy.yaml"
	bookinfoProductpagePolicyYaml = "testdata/bookinfo/platform/kube/rbac/productpage-policy.yaml"
	bookinfoReviewPolicyYaml 	= "testdata/bookinfo/platform/kube/rbac/details-reviews-policy.yaml"
	bookinfoRatingPolicyYaml 	= "testdata/bookinfo/platform/kube/rbac/ratings-policy.yaml"

	
	httpbinYaml					= "testdata/httpbin/httpbin.yaml"
	httpbinv1Yaml				= "testdata/httpbin/httpbin-v1.yaml"
	httpbinv2Yaml 				= "testdata/httpbin/httpbin-v2.yaml"
	httpbinLegacyYaml 			= "testdata/httpbin/httpbin-legacy.yaml"
	httpbinGatewayYaml			= "testdata/httpbin/networking/httpbin-gateway.yaml"
	httpbinGatewayv2Yaml		= "testdata/httpbin/networking/httpbin-gateway-2.yaml"
	httpbinGatewayHTTPSYaml		= "testdata/httpbin/networking/httpbin-gateway-https.yaml"
	httpbinGatewayHTTPSMutualYaml = "testdata/httpbin/networking/httpbin-gateway-https-mutual.yaml"
	httpbinRouteYaml			= "testdata/httpbin/networking/httpbin-route.yaml"
	httpbinRoutev2Yaml			= "testdata/httpbin/networking/httpbin-route-2.yaml"
	httpbinRouteHTTPSYaml		= "testdata/httpbin/networking/httpbin-route-https.yaml"
	httpbinOCPRouteYaml			= "testdata/httpbin/networking/httpbin-ocp-route.yaml"   // will be handled by maistra/ior
	httpbinOCPRouteHTTPSYaml	= "testdata/httpbin/networking/httpbin-ocp-route-https.yaml" // will be handled by maistra/ior
	httpbinTimeoutYaml			= "testdata/httpbin/networking/httpbin-ext-timeout.yaml"
	httpbinCircuitBreakerYaml 	= "testdata/httpbin/networking/httpbin-circuit-breaker.yaml"
	httpbinFortioYaml 			= "testdata/httpbin/sample-client/fortio-deploy.yaml"
	httpbinServiceYaml 			= "testdata/httpbin/networking/httpbin-service.yaml"
	httpbinAllv1Yaml 			= "testdata/httpbin/networking/virtual-service-httpbin-all-v1.yaml"
	httpbinMirrorv2Yaml 		= "testdata/httpbin/networking/virtual-service-httpbin-mirror-v2.yaml"	

	sleepYaml					= "testdata/sleep/sleep.yaml"
	sleepv2Yaml 				= "testdata/sleep/sleep-v2.yaml"
	sleepLegacyYaml 			= "testdata/sleep/sleep-legacy.yaml"
	sleepIPRangeYaml			= "testdata/sleep/sleep-ip-range.yaml"
	egressHTTPBinYaml 			= "testdata/egress/serviceEntry-httpbin.yaml"
	egressGoogleYaml			= "testdata/egress/serviceEntry-google.yaml"
	
	echoYaml 					= "testdata/tcp-echo/tcp-echo-services.yaml"
	echoAllv1Yaml 				= "testdata/tcp-echo/tcp-echo-all-v1.yaml"
	echo20v2Yaml 				= "testdata/tcp-echo/tcp-echo-20-v2.yaml"

	jwtAuthYaml					= "testdata/policy/jwt-auth.yaml"
	jwtURL 						= "https://raw.githubusercontent.com/istio/istio/release-1.0/security/tools/jwt/samples/demo.jwt"

	livenessHTTPYaml 			= "testdata/health-check/liveness-http.yaml"
	livenessCommandYaml 		= "testdata/health-check/liveness-command.yaml"

	tlsPolicyYaml 				= "testdata/policy/mutual_tls_policy.yaml"
	tlsRuleYaml 				= "testdata/policy/mutual_tls_destinationrule.yaml"
	mixerDenyPolicyYaml 		= "testdata/policy/mixer-rule-deny-label.yaml"
	policyCheckVersionYaml 		= "testdata/policy/checkversion-rule.yaml"
	policyAppversionYaml 		= "testdata/policy/appversion-instance.yaml"
	whitelistHandlerYaml 		= "testdata/policy/whitelist-handler.yaml"
	rateLimitYaml 				= "testdata/policy/mixer-rule-productpage-ratelimit.yaml"
	rateLimitConditionalYaml 	= "testdata/policy/mixer-rule-conditional.yaml"

	telemetryYaml 				= "testdata/telemetry/new_telemetry.yaml"
	telemetryTCPYaml 			= "testdata/telemetry/tcp_telemetry.yaml"
	loggingStackYaml 			= "testdata/telemetry/logging-stack.yaml"
	fluentdYaml 				= "testdata/telemetry/fluentd-istio.yaml"

	nginxYaml 					= "testdata/https/nginx-app.yaml"
	nginxNoSidecarYaml 			= "testdata/https/nginx-app-without-sidecar.yaml"

	httpbinSampleServerCertKey 	= "testdata/certs/httpbin.example.com/3_application/private/httpbin.example.com.key.pem"
	httpbinSampleServerCert 	= "testdata/certs/httpbin.example.com/3_application/certs/httpbin.example.com.cert.pem"
	httpbinSampleClientCertKey  = "testdata/certs/httpbin.example.com/4_client/private/httpbin.example.com.key.pem"
	httpbinSampleClientCert 	= "testdata/certs/httpbin.example.com/4_client/certs/httpbin.example.com.cert.pem"
	httpbinSampleCACert 		= "testdata/certs/httpbin.example.com/2_intermediate/certs/ca-chain.cert.pem"

	testNamespace				= "bookinfo"
	testUsername				= "jason"
	kubeconfigFile				= ""
)
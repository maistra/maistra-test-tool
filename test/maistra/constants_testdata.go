// Copyright 2019 Red Hat, Inc.
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

package maistra

import (
	"io/ioutil"
	"strings"
)

const (
	bookinfoYaml           = "testdata/bookinfo/platform/kube/bookinfo.yaml"
	bookinfoGateway        = "testdata/bookinfo/networking/bookinfo-gateway.yaml"
	bookinfoRuleAllYaml    = "testdata/bookinfo/networking/destination-rule-all.yaml"
	bookinfoRuleAllTLSYaml = "testdata/bookinfo/networking/destination-rule-all-mtls.yaml"

	bookinfoAllv1Yaml                   = "testdata/bookinfo/networking/virtual-service-all-v1.yaml"
	bookinfoRatingDelayYaml             = "testdata/bookinfo/networking/virtual-service-ratings-test-delay.yaml"
	bookinfoRatingDelayv2Yaml           = "testdata/bookinfo/networking/virtual-service-ratings-test-delay-2.yaml"
	bookinfoRatingAbortYaml             = "testdata/bookinfo/networking/virtual-service-ratings-test-abort.yaml"
	bookinfoRatingDBYaml                = "testdata/bookinfo/networking/virtual-service-ratings-db.yaml"
	bookinfoRatingMySQLYaml             = "testdata/bookinfo/networking/virtual-service-ratings-mysql.yaml"
	bookinfoRatingMySQLv2Yaml           = "testdata/bookinfo/networking/bookinfo-ratings-v2-mysql.yaml"
	bookinfoRatingMySQLServiceEntryYaml = "testdata/bookinfo/networking/bookinfo-ratings-mysql-service-entry.yaml"
	bookinfoReviewTestv2Yaml            = "testdata/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
	bookinfoReview50v3Yaml              = "testdata/bookinfo/networking/virtual-service-reviews-50-v3.yaml"
	bookinfoReviewv3Yaml                = "testdata/bookinfo/networking/virtual-service-reviews-v3.yaml"
	bookinfoReviewv2v3Yaml              = "testdata/bookinfo/networking/virtual-service-reviews-jason-v2-v3.yaml"
	bookinfoReviewTimeoutYaml           = "testdata/bookinfo/networking/virtual-service-reviews-timeout.yaml"
	bookinfoDBYaml                      = "testdata/bookinfo/platform/kube/bookinfo-db.yaml"
	bookinfoAddServiceAccountYaml       = "testdata/bookinfo/platform/kube/bookinfo-add-serviceaccount.yaml"
	bookinfoRBACOnTemplate              = "testdata/bookinfo/platform/kube/rbac/rbac-config-ON.yaml"
	bookinfoRBACOnDBTemplate            = "testdata/bookinfo/platform/kube/rbac/rbac-config-on-mongodb.yaml"
	bookinfoNamespacePolicyTemplate     = "testdata/bookinfo/platform/kube/rbac/namespace-policy.yaml"
	bookinfoProductpagePolicyTemplate   = "testdata/bookinfo/platform/kube/rbac/productpage-policy.yaml"
	bookinfoReviewPolicyTemplate        = "testdata/bookinfo/platform/kube/rbac/details-reviews-policy.yaml"
	bookinfoRatingPolicyTemplate        = "testdata/bookinfo/platform/kube/rbac/ratings-policy.yaml"
	bookinfoRatingv2ServiceAccount      = "testdata/bookinfo/platform/kube/rbac/ratings-v2-add-serviceaccount.yaml"
	bookinfoMongodbPolicyTemplate       = "testdata/bookinfo/platform/kube/rbac/mongodb-policy.yaml"
	bookinfoRatingv2Yaml                = "testdata/bookinfo/platform/kube/bookinfo-ratings-v2.yaml"

	bookinfoSampleServerCertKey = "testdata/certs/bookinfo.com/3_application/private/bookinfo.com.key.pem"
	bookinfoSampleServerCert    = "testdata/certs/bookinfo.com/3_application/certs/bookinfo.com.cert.pem"

	httpbinYaml                   = "testdata/httpbin/httpbin.yaml"
	httpbinv1Yaml                 = "testdata/httpbin/httpbin-v1.yaml"
	httpbinv2Yaml                 = "testdata/httpbin/httpbin-v2.yaml"
	httpbinLegacyYaml             = "testdata/httpbin/httpbin-legacy.yaml"
	httpbinGatewayYaml            = "testdata/httpbin/networking/httpbin-gateway.yaml"
	httpbinGatewayv2Yaml          = "testdata/httpbin/networking/httpbin-gateway-2.yaml"
	httpbinGatewayHTTPSYaml       = "testdata/httpbin/networking/httpbin-gateway-https.yaml"
	httpbinGatewayHTTPSMutualYaml = "testdata/httpbin/networking/httpbin-gateway-https-mutual.yaml"
	httpbinRouteYaml              = "testdata/httpbin/networking/httpbin-route.yaml"
	httpbinRoutev2Yaml            = "testdata/httpbin/networking/httpbin-route-2.yaml"
	httpbinRouteHTTPSYaml         = "testdata/httpbin/networking/httpbin-route-https.yaml"
	httpbinOCPRouteYaml           = "testdata/httpbin/networking/httpbin-ocp-route.yaml"       // will be handled by maistra/ior
	httpbinOCPRouteHTTPSYaml      = "testdata/httpbin/networking/httpbin-ocp-route-https.yaml" // will be handled by maistra/ior
	httpbinTimeoutYaml            = "testdata/httpbin/networking/httpbin-ext-timeout.yaml"
	httpbinCircuitBreakerYaml     = "testdata/httpbin/networking/httpbin-circuit-breaker.yaml"
	httpbinFortioYaml             = "testdata/httpbin/sample-client/fortio-deploy.yaml"
	httpbinServiceYaml            = "testdata/httpbin/networking/httpbin-service.yaml"
	httpbinAllv1Yaml              = "testdata/httpbin/networking/virtual-service-httpbin-all-v1.yaml"
	httpbinMirrorv2Yaml           = "testdata/httpbin/networking/virtual-service-httpbin-mirror-v2.yaml"
	httpbinPolicyAllYaml		  = "testdata/httpbin/httpbin-all.yaml"
	httpbinKeyvalTemplateYaml     = "testdata/httpbin/policy/keyval-template.yaml"
	httpbinKeyvalYaml             = "testdata/httpbin/policy/keyval.yaml"	

	sleepYaml         = "testdata/sleep/sleep.yaml"
	sleepv2Yaml       = "testdata/sleep/sleep-v2.yaml"
	sleepLegacyYaml   = "testdata/sleep/sleep-legacy.yaml"
	sleepIPRangeYaml  = "testdata/sleep/sleep-ip-range.yaml"
	egressHTTPBinYaml = "testdata/egress/serviceEntry-httpbin.yaml"
	egressGoogleYaml  = "testdata/egress/serviceEntry-google.yaml"

	echoYaml      = "testdata/tcp-echo/tcp-echo-services.yaml"
	echoAllv1Yaml = "testdata/tcp-echo/tcp-echo-all-v1.yaml"
	echo20v2Yaml  = "testdata/tcp-echo/tcp-echo-20-v2.yaml"

	jwtAuthYaml = "testdata/policy/jwt-auth.yaml"
	jwtURL      = "https://raw.githubusercontent.com/istio/istio/release-1.1/security/tools/jwt/samples/demo.jwt"
	jwtURLGroup = "https://raw.githubusercontent.com/istio/istio/release-1.1/security/tools/jwt/samples/groups-scope.jwt"
	jwtGen      = "testdata/security/gen-jwt.py"
	jwtKey      = "testdata/security/key.pem"

	livenessHTTPYaml    = "testdata/health-check/liveness-http.yaml"
	livenessCommandYaml = "testdata/health-check/liveness-command.yaml"

	tlsPolicyYaml            = "testdata/policy/mutual_tls_policy.yaml"
	tlsRuleYaml              = "testdata/policy/mutual_tls_destinationrule.yaml"
	mixerDenyPolicyYaml      = "testdata/policy/mixer-rule-deny-label.yaml"
	policyDenyWhitelistYaml  = "testdata/policy/mixer-rule-deny-whitelist.yaml"
	policyDenyIPYaml         = "testdata/policy/mixer-rule-deny-ip.yaml"
	rateLimitYamlTemplate    = "testdata/policy/mixer-rule-productpage-ratelimit.yaml"
	rateLimitConditionalYaml = "testdata/policy/mixer-rule-conditional.yaml"

	telemetryYaml    = "testdata/telemetry/new_telemetry.yaml"
	telemetryTCPYaml = "testdata/telemetry/tcp_telemetry.yaml"
	loggingStackYaml = "testdata/telemetry/logging-stack.yaml"
	fluentdYaml      = "testdata/telemetry/fluentd-istio.yaml"

	nginxYaml          = "testdata/https/nginx-app.yaml"
	nginxNoSidecarYaml = "testdata/https/nginx-app-without-sidecar.yaml"
	nginxConf          = "testdata/https/default.conf"

	httpbinSampleServerCertKey = "testdata/certs/httpbin.example.com/3_application/private/httpbin.example.com.key.pem"
	httpbinSampleServerCert    = "testdata/certs/httpbin.example.com/3_application/certs/httpbin.example.com.cert.pem"
	httpbinSampleClientCertKey = "testdata/certs/httpbin.example.com/4_client/private/httpbin.example.com.key.pem"
	httpbinSampleClientCert    = "testdata/certs/httpbin.example.com/4_client/certs/httpbin.example.com.cert.pem"
	httpbinSampleCACert        = "testdata/certs/httpbin.example.com/2_intermediate/certs/ca-chain.cert.pem"

	caCert 				= "testdata/certs/ca-cert.pem"
	caCertKey 			= "testdata/certs/ca-key.pem"
	caRootCert			= "testdata/certs/root-cert.pem"
	caCertChain 		= "testdata/certs/cert-chain.pem"

	testNamespace  = "bookinfo"
	testUsername   = "jason"
	kubeconfigFile = ""

	meshNamespace = "istio-system"

	testRetryTimes = 5
)

var (
	bookinfoRBACOn            string
	bookinfoRBAConDB          string
	bookinfoNamespacePolicy   string
	bookinfoProductpagePolicy string
	bookinfoReviewPolicy      string
	bookinfoRatingPolicy      string
	bookinfoMongodbPolicy     string
	rateLimitYaml			  string
)


func updateYaml() {
	data, _ := ioutil.ReadFile(bookinfoRBACOnTemplate)
	bookinfoRBACOn = strings.Replace(string(data), "\"default\"", "\""+testNamespace+"\"", -1)

	data, _ = ioutil.ReadFile(bookinfoRBACOnDBTemplate)
	bookinfoRBAConDB = strings.Replace(string(data), "mongodb.default", "mongodb."+testNamespace, -1)

	data, _ = ioutil.ReadFile(bookinfoNamespacePolicyTemplate)
	bookinfoNamespacePolicytmp := strings.Replace(string(data), "default", testNamespace, -1)
	bookinfoNamespacePolicy = strings.Replace(bookinfoNamespacePolicytmp, "[mesh]", meshNamespace, -1)

	data, _ = ioutil.ReadFile(bookinfoProductpagePolicyTemplate)
	bookinfoProductpagePolicy = strings.Replace(string(data), "default", testNamespace, -1)

	data, _ = ioutil.ReadFile(bookinfoReviewPolicyTemplate)
	bookinfoReviewPolicy = strings.Replace(string(data), "default", testNamespace, -1)

	data, _ = ioutil.ReadFile(bookinfoRatingPolicyTemplate)
	bookinfoRatingPolicy = strings.Replace(string(data), "default", testNamespace, -1)

	data, _ = ioutil.ReadFile(bookinfoMongodbPolicyTemplate)
	bookinfoMongodbPolicy = strings.Replace(string(data), "default", testNamespace, -1)

	data, _ = ioutil.ReadFile(rateLimitYamlTemplate)
	rateLimitYaml = strings.Replace(string(data), "[mesh]", meshNamespace, -1)
	rateLimitYaml = strings.Replace(rateLimitYaml, "[test]", testNamespace, -1)

}

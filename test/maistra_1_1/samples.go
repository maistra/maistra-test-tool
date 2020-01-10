// Copyright 2020 Red Hat, Inc.
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

package main

const (
	bookinfoYaml 			= "samples/bookinfo/platform/kube/bookinfo.yaml"
	bookinfoGateway			= "samples/bookinfo/networking/bookinfo-gateway.yaml"
	bookinfoRuleAllYaml    	= "samples/bookinfo/networking/destination-rule-all.yaml"
	bookinfoRuleAllTLSYaml 	= "samples/bookinfo/networking/destination-rule-all-mtls.yaml"
	bookinfoAllv1Yaml       = "samples/bookinfo/networking/virtual-service-all-v1.yaml"
	bookinfoReviewV2Yaml    = "samples/bookinfo/networking/virtual-service-reviews-test-v2.yaml"

	bookinfoDBYaml          = "samples/bookinfo/platform/kube/bookinfo-db.yaml"

	httpbinYaml             = "samples/httpbin/httpbin.yaml"
	httpbinFortioYaml       = "samples/httpbin/sample-client/fortio-deploy.yaml"

	echoYaml				= "samples/tcp-echo/tcp-echo-services.yaml"

	sleepYaml				= "samples/sleep/sleep.yaml"

	nginxYaml          		= "samples/https/nginx-app.yaml"
	nginxNoSidecarYaml 		= "samples/https/nginx-app-without-sidecar.yaml"
	

	kubeconfig 				= ""
	testNamespace 			= "bookinfo"
	testUsername   			= "jason"
	gatewayHTTP 			= "localhost:8001/api/v1/namespaces/istio-system/services/istio-ingressgateway:80/proxy"
	
)



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

package egress

import (
	_ "embed"
	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

var (
	nginxClientCertKey = env.GetRootDir() + "/sampleCerts/nginx.example.com/nginx-client.example.com.key"
	nginxClientCert    = env.GetRootDir() + "/sampleCerts/nginx.example.com/nginx-client.example.com.crt"
	nginxServerCACert  = env.GetRootDir() + "/sampleCerts/nginx.example.com/example.com.crt"
)

var (
	meshNamespace = env.GetDefaultMeshNamespace()
	smcp          = ossm.DefaultSMCP()

	//go:embed yaml/external-httpbin.yaml
	httpbinServiceEntry string

	//go:embed yaml/external-nginx.yaml
	nginxServiceEntry string

	//go:embed yaml/external-httpbin-http-gateway.yaml
	httpbinHttpGateway string

	//go:embed yaml/external-nginx-tls-passthrough-gateway.yaml
	nginxTlsPassthroughGateway string

	//go:embed yaml/external-nginx-tls-istio-mutual-gateway.yaml
	nginxTlsIstioMutualGateway string

	//go:embed yaml/mesh-route-http-requests-to-https-port.yaml
	meshRouteHttpRequestsToHttpsPort string

	//go:embed yaml/originate-tls-to-nginx.yaml
	originateTlsToNginx string

	//go:embed yaml/originate-mtls-to-nginx.yaml
	originateMtlsToNginx string

	//go:embed yaml/originate-mtls-sds-to-nginx.yaml
	originateMtlsSdsSToNginx string
)

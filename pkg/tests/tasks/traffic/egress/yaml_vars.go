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
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
)

var (
	nginxClientCertKey = env.GetRootDir() + "/sampleCerts/nginx.example.com/nginx-client.example.com.key"
	nginxClientCert    = env.GetRootDir() + "/sampleCerts/nginx.example.com/nginx-client.example.com.crt"
	nginxServerCACert  = env.GetRootDir() + "/sampleCerts/nginx.example.com/example.com.crt"
)

var (
	smcpName      = env.Getenv("SMCPNAME", "basic")
	meshNamespace = env.Getenv("MESHNAMESPACE", "istio-system")
	smcp          = template.SMCP{
		Name:      smcpName,
		Namespace: meshNamespace,
		Rosa:      env.IsRosa()}
)

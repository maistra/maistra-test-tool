// Copyright Red Hat, Inc.
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

package examples

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

var (
	nginxServerCertKey   = fmt.Sprintf("%s/sampleCerts/nginx.example.com/nginx.example.com.key", env.GetRootDir())
	nginxServerCert      = fmt.Sprintf("%s/sampleCerts/nginx.example.com/nginx.example.com.crt", env.GetRootDir())
	nginxServerCACert    = fmt.Sprintf("%s/sampleCerts/nginx.example.com/example.com.crt", env.GetRootDir())
	meshExtServerCertKey = fmt.Sprintf("%s/sampleCerts/nginx.example.com/my-nginx.mesh-external.svc.cluster.local.key", env.GetRootDir())
	meshExtServerCert    = fmt.Sprintf("%s/sampleCerts/nginx.example.com/my-nginx.mesh-external.svc.cluster.local.crt", env.GetRootDir())

	nginxConf     = fmt.Sprintf("%s/testdata/examples/common/nginx/nginx.conf", env.GetRootDir())
	nginxConfMTls = fmt.Sprintf("%s/testdata/examples/common/nginx/nginx_mesh_external_ssl.conf", env.GetRootDir())
	nginxYaml     = fmt.Sprintf("%s/testdata/examples/common/nginx/nginx.yaml", env.GetRootDir())
)

// TODO: remove these functions when the refactoring is done

func NginxYamlFile() string {
	return nginxYaml
}

func NginxConfMTlsFile() string {
	return nginxConfMTls
}

func NginxConfFile() string {
	return nginxConf
}

func NginxServerCertKey() string {
	return nginxServerCertKey
}

func NginxServerCert() string {
	return nginxServerCert
}

func NginxServerCACert() string {
	return nginxServerCACert
}

func MeshExtServerCertKey() string {
	return meshExtServerCertKey
}

func MeshExtServerCert() string {
	return meshExtServerCert
}

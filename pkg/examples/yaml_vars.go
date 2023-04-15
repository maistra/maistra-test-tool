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
	"os"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

var (
	branch  = getenv("SAMPLEARCH", "x86")
	basedir = env.GetRootDir() + "/testdata/examples"
	certdir = env.GetRootDir() + "/sampleCerts"
)

var (
	bookinfoYaml           = fmt.Sprintf("%s/%s/bookinfo/bookinfo.yaml", basedir, branch)
	bookinfoGateway        = fmt.Sprintf("%s/%s/bookinfo/bookinfo-gateway.yaml", basedir, branch)
	bookinfoRuleAllYaml    = fmt.Sprintf("%s/%s/bookinfo/destination-rule-all.yaml", basedir, branch)
	bookinfoRuleAllTLSYaml = fmt.Sprintf("%s/%s/bookinfo/destination-rule-all-mtls.yaml", basedir, branch)
	BookinfoRatingsV2Yaml  = fmt.Sprintf("%s/%s/bookinfo/bookinfo-ratings-v2.yaml", basedir, branch)
	BookinfoDBYaml         = fmt.Sprintf("%s/%s/bookinfo/bookinfo-db.yaml", basedir, branch)

	BookinfoVirtualServiceReviewsV3Yaml = fmt.Sprintf("%s/common/bookinfo/virtual-service-reviews-v3.yaml", basedir)

	nginxServerCertKey   = fmt.Sprintf("%s/nginx.example.com/nginx.example.com.key", certdir)
	nginxServerCert      = fmt.Sprintf("%s/nginx.example.com/nginx.example.com.crt", certdir)
	nginxServerCACert    = fmt.Sprintf("%s/nginx.example.com/example.com.crt", certdir)
	meshExtServerCertKey = fmt.Sprintf("%s/nginx.example.com/my-nginx.mesh-external.svc.cluster.local.key", certdir)
	meshExtServerCert    = fmt.Sprintf("%s/nginx.example.com/my-nginx.mesh-external.svc.cluster.local.crt", certdir)
	nginxConf            = fmt.Sprintf("%s/%s/nginx/nginx.conf", basedir, branch)
	nginxConfMTls        = fmt.Sprintf("%s/%s/nginx/nginx_mesh_external_ssl.conf", basedir, branch)
	nginxYaml            = fmt.Sprintf("%s/%s/nginx/nginx.yaml", basedir, branch)
)

// TODO: remove these functions when the refactoring is done

func BookinfoYamlFile() string {
	return bookinfoYaml
}

func BookinfoGatewayYamlFile() string {
	return bookinfoGateway
}

func BookinfoRuleAllYamlFile() string {
	return bookinfoRuleAllYaml
}

func BookinfoRuleAllMTLSYamlFile() string {
	return bookinfoRuleAllTLSYaml
}

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

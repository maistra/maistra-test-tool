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

	echoYaml      = fmt.Sprintf("%s/%s/tcp-echo/tcp-echo-services.yaml", basedir, branch)
	echoWithProxy = fmt.Sprintf("%s/%s/tcp-echo/tcp-echo.yaml", basedir, branch)
	echov1Yaml    = fmt.Sprintf("%s/%s/tcp-echo/tcp-echo-v1.yaml", basedir, branch)
	echov2Yaml    = fmt.Sprintf("%s/%s/tcp-echo/tcp-echo-v2.yaml", basedir, branch)

	fortioYaml = fmt.Sprintf("%s/%s/httpbin/sample-client/fortio-deploy.yaml", basedir, branch)

	httpbinYaml       = fmt.Sprintf("%s/%s/httpbin/httpbin.yaml", basedir, branch)
	httpbinLegacyYaml = fmt.Sprintf("%s/%s/httpbin/httpbin_legacy.yaml", basedir, branch)
	httpbinv1Yaml     = fmt.Sprintf("%s/%s/httpbin/httpbinv1.yaml", basedir, branch)
	httpbinv2Yaml     = fmt.Sprintf("%s/%s/httpbin/httpbinv2.yaml", basedir, branch)

	nginxServerCertKey   = fmt.Sprintf("%s/nginx.example.com/nginx.example.com.key", certdir)
	nginxServerCert      = fmt.Sprintf("%s/nginx.example.com/nginx.example.com.crt", certdir)
	nginxServerCACert    = fmt.Sprintf("%s/nginx.example.com/example.com.crt", certdir)
	meshExtServerCertKey = fmt.Sprintf("%s/nginx.example.com/my-nginx.mesh-external.svc.cluster.local.key", certdir)
	meshExtServerCert    = fmt.Sprintf("%s/nginx.example.com/my-nginx.mesh-external.svc.cluster.local.crt", certdir)
	nginxConf            = fmt.Sprintf("%s/%s/nginx/nginx.conf", basedir, branch)
	nginxYaml            = fmt.Sprintf("%s/%s/nginx/nginx.yaml", basedir, branch)

	redisYaml = fmt.Sprintf("%s/%s/redis/redis.yaml", basedir, branch)

	sleepYaml       = fmt.Sprintf("%s/%s/sleep/sleep.yaml", basedir, branch)
	sleepLegacyYaml = fmt.Sprintf("%s/%s/sleep/sleep_legacy.yaml", basedir, branch)
)

// TODO: remove these functions when the refactoring is done

func EchoYamlFile() string {
	return echoWithProxy
}

func EchoV1YamlFile() string {
	return echov1Yaml
}

func EchoV2YamlFile() string {
	return echov2Yaml
}

func HttpbinYamlFile() string {
	return httpbinYaml
}

func HttpbinLegacyYamlFile() string {
	return httpbinLegacyYaml
}

func HttpbinV1YamlFile() string {
	return httpbinv1Yaml
}

func HttpbinV2YamlFile() string {
	return httpbinv2Yaml
}

func SleepYamlFile() string {
	return sleepYaml
}

func SleepLegacyYamlFile() string {
	return sleepLegacyYaml
}

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

func FortioYamlFile() string {
	return fortioYaml
}

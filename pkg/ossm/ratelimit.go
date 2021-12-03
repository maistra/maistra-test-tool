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

package ossm

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

const (
	rateLimitFilterYaml_template = "../testdata/resources/yaml/ratelimit-envoyfilter_template.yaml"
	rateLimitFilterYaml          = "../testdata/resources/yaml/ratelimit-envoyfilter.yaml"
)

func cleanupRateLimiting(redisDeploy examples.Redis, bookinfoDeploy examples.Bookinfo) {
	util.Shell(`kubectl -n %s patch smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/techPreview/rateLimiting"}]'`, meshNamespace, smcpName)
	util.KubeDelete(meshNamespace, rateLimitSMCPPatch)
	redisDeploy.Uninstall()
	bookinfoDeploy.Uninstall()
}

func TestRateLimiting(t *testing.T) {
	redisDeploy := examples.Redis{Namespace: "redis"}
	bookinfo := examples.Bookinfo{Namespace: "bookinfo"}
	bookinfo.Install(false)

	defer cleanupRateLimiting(redisDeploy, bookinfo)

	if err := redisDeploy.Install(); err != nil {
		t.Fatal(err)
	}
	if _, err := util.Shell(`kubectl -n %s patch smcp/%s --type=merge --patch="%s"`, meshNamespace, smcpName, rateLimitSMCPPatch); err != nil {
		t.Fatal(err)
	}

	if _, err := util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName); err != nil {
		t.Fatal(err)
	}

	if err := util.CheckPodRunning(meshNamespace, "app=rls"); err != nil {
		t.Fatalf("rls deployment not ready: %v", err)
	}
	util.Shell(`envsubst < %s > %s`, rateLimitFilterYaml_template, rateLimitFilterYaml)
	if err := util.KubeApply(meshNamespace, rateLimitFilterYaml); err != nil {
		t.Fatalf("error applying envoy filter: %v", err)
	}
	util.Shell(`kubectl -n %s get envoyfilter -o yaml > rrr.yaml`, meshNamespace)
	//util.Log.Info(msg)

	// Give some time to envoy filters apply
	time.Sleep(time.Second * 5)

	host, err := util.Shell("oc -n %s get route istio-ingressgateway -o jsonpath='{.spec.host}'", meshNamespace)
	if err != nil {
		t.Fatalf("error getting route hostname: %v", err)
	}
	host = strings.Trim(host, "'")

	time.Sleep(time.Duration(20) * time.Second)

	// Should work first time
	checkProductPageResponseCode(t, host, "200")

	// Should fail first time
	checkProductPageResponseCode(t, host, "429")

	// Should work again after 1 minute
	time.Sleep(time.Second * 65)
	checkProductPageResponseCode(t, host, "200")

	// Should fail
	time.Sleep(time.Second * 5)
	checkProductPageResponseCode(t, host, "429")
}

func checkProductPageResponseCode(t *testing.T, host string, expectedCode string) {
	t.Helper()

	code, err := util.Shell("curl -s -o /dev/null -w '%%{http_code}' http://%s/productpage", host)
	if err != nil {
		t.Fatalf("error getting productpage: %v", err)
	}
	code = strings.Trim(code, "'")
	if code != expectedCode {
		t.Fatalf("expected status code %q got %q", expectedCode, code)
	}
}

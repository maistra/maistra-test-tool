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
	rateLimitFilterYaml_template = `
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: filter-ratelimit
spec:
  workloadSelector:
    # select by label in the same namespace
    labels:
      istio: ingressgateway
  configPatches:
    # The Envoy config you want to modify
    - applyTo: HTTP_FILTER
      match:
        context: GATEWAY
        listener:
          filterChain:
            filter:
              name: "envoy.filters.network.http_connection_manager"
              subFilter:
                name: "envoy.filters.http.router"
      patch:
        operation: INSERT_BEFORE
        # Adds the Envoy Rate Limit Filter in HTTP filter chain.
        value:
          name: envoy.filters.http.ratelimit
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
            # domain can be anything! Match it to the ratelimter service config
            domain: productpage-ratelimit
            failure_mode_deny: true
            timeout: 10s
            rate_limit_service:
              grpc_service:
                envoy_grpc:
                  cluster_name: rate_limit_cluster
              transport_api_version: V3
    - applyTo: CLUSTER
      match:
        cluster:
          service: rls-{{ .Name }}.{{ .Namespace }}.svc.cluster.local
      patch:
        operation: ADD
        # Adds the rate limit service cluster for rate limit service defined in step 1.
        value:
          name: rate_limit_cluster
          type: STRICT_DNS
          connect_timeout: 10s
          lb_policy: ROUND_ROBIN
          http2_protocol_options: {}
          load_assignment:
            cluster_name: rate_limit_cluster
            endpoints:
            - lb_endpoints:
              - endpoint:
                  address:
                     socket_address:
                      address: rls-{{ .Name }}.{{ .Namespace }}.svc.cluster.local
                      port_value: 8081

---

apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: filter-ratelimit-svc
spec:
  workloadSelector:
    labels:
      istio: ingressgateway
  configPatches:
    - applyTo: VIRTUAL_HOST
      match:
        context: GATEWAY
        routeConfiguration:
          vhost:
            name: ""
            route:
              action: ANY
      patch:
        operation: MERGE
        # Applies the rate limit rules.
        value:
          rate_limits:
            - actions: # any actions in here
              - request_headers:
                  header_name: ":path"
                  descriptor_key: "PATH"	
`
)

func cleanupRateLimiting(redisDeploy examples.Redis, bookinfoDeploy examples.Bookinfo) {
	util.Shell(`kubectl -n %s patch smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/techPreview/rateLimiting"}]'`, meshNamespace, smcpName)
	util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName)
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(rateLimitFilterYaml_template, smcp))
	time.Sleep(time.Second * 5)
	util.KubeDeleteContents(meshNamespace, rateLimitSMCPPatch)
	util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName)
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

	if err := util.KubeApplyContents(meshNamespace, util.RunTemplate(rateLimitFilterYaml_template, smcp)); err != nil {
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
	time.Sleep(time.Second * 15)
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

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

package ossm_federation

import (
	_ "embed"
	"fmt"
	"math"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/curl"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestFederation(t *testing.T) {
	NewTest(t).Id("T31").Groups(Full, ARM).Run(func(t TestHelper) {
		//This test will be executed in multicluster only if two kubeconfigs are provided and those cluster are not in ROSA
		//To ROSA test of federation we have this testcase: TestMultiClusterFederationFailover
		t.Log("Both OC clusters are set respectively to cluster west and east if they are provided with two kubeconfig files, if not, both are set from the same file")
		ocWest, ocEast := setKubeconfig()

		federationTest{
			testdataPath: "testdata/traffic-splitting",
			west: config{
				oc:                ocWest,
				smcpName:          "west-mesh",
				smcpNamespace:     "west-mesh-system",
				bookinfoNamespace: "west-mesh-bookinfo",
			},
			east: config{
				oc:                ocEast,
				smcpName:          "east-mesh",
				smcpNamespace:     "east-mesh-system",
				bookinfoNamespace: "east-mesh-bookinfo",
			},
			controlPlaneInstaller: func(t TestHelper, ft federationTest, ingressServiceType string) {
				installSMCPandSMMR(t, ft.west, ft.testdataPath+"/west-mesh/smcp.yaml", ft.testdataPath+"/west-mesh/smmr.yaml", ingressServiceType)
				installSMCPandSMMR(t, ft.east, ft.testdataPath+"/east-mesh/smcp.yaml", ft.testdataPath+"/east-mesh/smmr.yaml", ingressServiceType)
			},
			bookinfoInstaller: defaultBookinfoInstaller,
			checker:           defaultChecker,
		}.run(t)
	})
}

func TestFederationDifferentCerts(t *testing.T) {
	NewTest(t).Id("T32").Groups(Full).Run(func(t TestHelper) {

		ocWest, ocEast := setKubeconfig()
		federationTest{
			testdataPath: "testdata/traffic-splitting",
			west: config{
				oc:                ocWest,
				smcpName:          "west-mesh",
				smcpNamespace:     "west-mesh-system",
				bookinfoNamespace: "west-mesh-bookinfo",
			},
			east: config{
				oc:                ocEast,
				smcpName:          "east-mesh",
				smcpNamespace:     "east-mesh-system",
				bookinfoNamespace: "east-mesh-bookinfo",
			},
			controlPlaneInstaller: func(t TestHelper, ft federationTest, ingressServiceType string) {
				t.Log("Create Secret 'cacerts' for custom CA certs in west-mesh")
				ft.west.oc.CreateGenericSecretFromFiles(t, ft.west.smcpNamespace, "cacerts",
					"testdata/cacerts/ca-cert.pem",
					"testdata/cacerts/ca-key.pem",
					"testdata/cacerts/root-cert.pem",
					"testdata/cacerts/cert-chain.pem")

				installSMCPandSMMR(t, ft.west, ft.testdataPath+"/west-mesh/smcp_custom_cert.yaml", ft.testdataPath+"/west-mesh/smmr.yaml", ingressServiceType)
				installSMCPandSMMR(t, ft.east, ft.testdataPath+"/east-mesh/smcp.yaml", ft.testdataPath+"/east-mesh/smmr.yaml", ingressServiceType)
			},
			bookinfoInstaller: defaultBookinfoInstaller,
			checker:           defaultChecker,
		}.run(t)
	})
}

func setKubeconfig() (*oc.OC, *oc.OC) {
	ocWest := oc.DefaultOC
	ocEast := oc.DefaultOC
	kubeconfig2 := env.GetKubeconfig2()
	if kubeconfig2 != "" && !env.IsRosa() {
		ocEast = oc.WithKubeconfig(kubeconfig2)
	}
	return ocWest, ocEast
}

func defaultBookinfoInstaller(t TestHelper, ft federationTest) {
	t.LogStep("Install ratings-v2 and mongodb in west-mesh")
	ft.west.oc.ApplyFile(t, ft.west.bookinfoNamespace, ft.testdataPath+"/west-mesh/bookinfo-ratings-service.yaml")
	ft.west.oc.ApplyTemplateString(t, ft.west.bookinfoNamespace, app.BookinfoRatingsV2Template, nil)
	ft.west.oc.ApplyTemplateString(t, ft.west.bookinfoNamespace, app.BookinfoDBTemplate, nil)
	ft.west.oc.ApplyString(t, ft.west.bookinfoNamespace, app.BookinfoRuleAll)

	t.LogStep("Install full bookinfo in east-mesh")
	ft.east.oc.ApplyTemplateString(t, ft.east.bookinfoNamespace, app.BookinfoTemplate, nil)          // install base bookinfo services
	ft.east.oc.ApplyTemplateString(t, ft.east.bookinfoNamespace, app.BookinfoRatingsV2Template, nil) // install ratings-v2
	ft.east.oc.ApplyString(t, ft.east.bookinfoNamespace, app.BookinfoGateway)                        // install gateway
	ft.east.oc.ApplyString(t, ft.east.bookinfoNamespace, app.BookinfoRuleAll)
	ft.east.oc.ApplyString(t, ft.east.bookinfoNamespace, app.BookinfoVirtualServiceReviewsV3) // reviews always go to reviews-v3
	ft.east.oc.ApplyFile(t, ft.east.bookinfoNamespace, ft.testdataPath+"/east-mesh/mongodb-service.yaml")
	ft.east.oc.ApplyFile(t, ft.east.bookinfoNamespace, ft.testdataPath+"/east-mesh/mongodb-remote-virtualservice.yaml") // mongodb always goes to west-mesh
	ft.east.oc.ApplyFile(t, ft.east.bookinfoNamespace, ft.testdataPath+"/east-mesh/ratings-split-virtualservice.yaml")  // 50-50 split between local ratings and ratings in west-mesh
}

func defaultChecker(t TestHelper, ft federationTest) {
	t.LogStep("Check if traffic is split between ratings-v1 in east-mesh and west-mesh")
	retry.UntilSuccess(t, func(t TestHelper) {
		t.LogStep("Check if east-mesh can see services from west-mesh")
		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(300), func(t TestHelper) {
			ft.east.oc.Invoke(t,
				`oc -n east-mesh-system get importedservicesets west-mesh -o json`,
				assert.OutputContains("mongodb.bookinfo.svc.east-mesh-exports.local",
					"mongodb service from west-mesh successfully imported in east-mesh",
					"mongodb service from west-mesh not imported in east-mesh"),
				assert.OutputContains("ratings.bookinfo.svc.east-mesh-exports.local",
					"ratings service from west-mesh successfully imported in east-mesh",
					"ratings service from west-mesh not imported in east-mesh"))
		})

		eastMeshProductPageURL := fmt.Sprintf("http://%s/productpage", ft.east.oc.GetRouteURL(t, ft.east.smcpNamespace, "istio-ingressgateway"))

		eastCount0 := getRatingsV2RequestCount(t, ft.east)
		westCount0 := getRatingsV2RequestCount(t, ft.west)
		for i := 0; i < 10; i++ {
			curl.Request(t, eastMeshProductPageURL, nil)
		}
		numberOfRequestsEast := getRatingsV2RequestCount(t, ft.east) - eastCount0
		numberOfRequestsWest := getRatingsV2RequestCount(t, ft.west) - westCount0

		if numberOfRequestsEast == 0 {
			t.Fatal("no request received by ratings-v2 in east-mesh")
		} else {
			t.LogSuccessf("ratings-v2 in east-mesh received %d requests", numberOfRequestsEast)
		}

		if numberOfRequestsWest == 0 {
			t.Fatal("no request received by ratings-v2 in west-mesh")
		} else {
			t.LogSuccessf("ratings-v2 in west-mesh received %d requests", numberOfRequestsWest)
		}
	})

	// TODO: check that the number of connections received by mongodb matches the number of requests
}

func getRatingsV2RequestCount(t TestHelper, c config) int {
	t.T().Helper()
	metrics := istio.GetProxyMetrics(t, c.oc,
		pod.MatchingSelector("app=ratings,version=v2", c.bookinfoNamespace),
		"istio_requests_total",
		"destination_workload=ratings-v2", "reporter=destination")

	count := 0
	for _, m := range metrics {
		count += int(math.Round(*m.Counter.Value))
	}
	return count
}

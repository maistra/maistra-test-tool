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
	"strings"
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

// use the following command to monitor the test: watch "kubecolor get smcp -n west-mesh-system;echo;kubecolor get pods -n west-mesh-system;echo;kubecolor get pods -n west-mesh-bookinfo; echo; echo '========================================================================='; echo; kubecolor get smcp -n east-mesh-system;echo;kubecolor get pods -n east-mesh-system;echo;kubecolor get pods -n east-mesh-bookinfo"

func TestMultiClusterFederationFailover(t *testing.T) {
	NewTest(t).Groups(Full).Run(func(t TestHelper) {
		kubeconfig2 := env.Getenv("KUBECONFIG2", "")
		if kubeconfig2 == "" {
			t.Skip("this test only runs when the KUBECONFIG2 environment variable points to the kubeconfig of the second cluster")
		}

		ocWest := oc.DefaultOC
		ocEast := oc.WithKubeconfig(kubeconfig2)

		westRegion, westZone := getRegionAndZone(t, ocWest)
		eastRegion, eastZone := getRegionAndZone(t, ocEast)
		if eastRegion == westRegion {
			t.Fatalf("KUBECONFIG and KUBECONFIG2 must point to clusters in different regions, but they are both in %s", westRegion)
		}

		federationTest{
			testdataPath: "testdata/failover",
			west: config{
				oc:                ocWest,
				smcpName:          "west-mesh",
				smcpNamespace:     "west-mesh-system",
				bookinfoNamespace: "bookinfo-ha",
				region:            westRegion,
				zone:              westZone,
			},
			east: config{
				oc:                ocEast,
				smcpName:          "east-mesh",
				smcpNamespace:     "east-mesh-system",
				bookinfoNamespace: "bookinfo-ha",
				region:            eastRegion,
				zone:              eastZone,
			},
			controlPlaneInstaller: func(t TestHelper, ft federationTest) {
				installSMCPandSMMR(t, ft.west, ft.testdataPath+"/west-mesh/smcp.yaml", ft.testdataPath+"/west-mesh/smmr.yaml")
				installSMCPandSMMR(t, ft.east, ft.testdataPath+"/east-mesh/smcp.yaml", ft.testdataPath+"/east-mesh/smmr.yaml")
			},
			bookinfoInstaller: func(t TestHelper, ft federationTest) {
				t.LogStep("Install bookinfo in west-mesh")
				ft.west.oc.ApplyTemplateString(t, ft.west.bookinfoNamespace, app.BookinfoTemplate, nil)
				ft.west.oc.ApplyString(t, ft.west.bookinfoNamespace, app.BookinfoRuleAll)

				t.LogStep("Install bookinfo in east-mesh")
				ft.east.oc.ApplyTemplateString(t, ft.east.bookinfoNamespace, app.BookinfoTemplate, nil)
				ft.east.oc.ApplyString(t, ft.east.bookinfoNamespace, app.BookinfoGateway)
				ft.east.oc.ApplyString(t, ft.east.bookinfoNamespace, app.BookinfoRuleAll)
				ft.east.oc.ApplyString(t, ft.east.bookinfoNamespace, app.BookinfoVirtualServiceReviewsV3)

				t.LogStep("Install fail-over DestinationRule for ratings service in east-mesh")
				ft.east.oc.ApplyTemplateFile(t, ft.east.bookinfoNamespace, ft.testdataPath+"/east-mesh/destinationrule-failover.yaml", map[string]string{
					"EastMeshRegion": eastRegion,
					"WestMeshRegion": westRegion,
				})
			},
			checker: func(t TestHelper, ft federationTest) {
				t.LogStep("Check if east-mesh can see services from west-mesh")
				retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(300), func(t TestHelper) {
					ft.east.oc.Invoke(t,
						`oc -n east-mesh-system get importedservicesets west-mesh -o json`,
						assert.OutputContains("ratings.bookinfo.svc.east-mesh-exports.local",
							"ratings service from west-mesh successfully imported in east-mesh",
							"ratings service from west-mesh not imported in east-mesh"))
				})

				eastMeshProductPageURL := fmt.Sprintf("http://%s/productpage", ocEast.GetRouteURL(t, ft.east.smcpNamespace, "istio-ingressgateway"))

				t.LogStep("Send HTTP request to east-mesh, expecting ratings-v1 in east-mesh to receive ratings request and not west-mesh")
				assertRatingsInEastMeshReceivesRequest(t, eastMeshProductPageURL, ft)

				t.LogStep("Scale Deployment ratings-v1 in east-mesh to zero replicas in order to trigger failover to west-mesh")
				ocEast.ScaleDeploymentAndWait(t, ft.east.bookinfoNamespace, "ratings-v1", 0)

				t.LogStep("Send HTTP request to east-mesh, expecting ratings-v1 in west-mesh to receive ratings request")
				assertRatingsInWestMeshReceivesRequest(t, eastMeshProductPageURL, ft)

				t.LogStep("Scale Deployment ratings-v1 in east-mesh back to one replica in order to disable failover again")
				ocEast.ScaleDeploymentAndWait(t, ft.east.bookinfoNamespace, "ratings-v1", 1)

				t.LogStep("Send HTTP request to east-mesh, expecting ratings-v1 in east-mesh to receive ratings request and not west-mesh")
				assertRatingsInEastMeshReceivesRequest(t, eastMeshProductPageURL, ft)
			},
		}.run(t)
	})
}

func getRegionAndZone(t TestHelper, oc *oc.OC) (string, string) {
	output := oc.Invoke(t, "oc get nodes -o jsonpath='{.items[0].metadata.labels.topology\\.kubernetes\\.io/region} {.items[0].metadata.labels.topology\\.kubernetes\\.io/region}'")
	arr := strings.Split(output, " ")
	return arr[0], arr[1]
}

func assertRatingsInEastMeshReceivesRequest(t TestHelper, eastMeshBookinfoURL string, ft federationTest) {
	retry.UntilSuccess(t, func(t TestHelper) {
		eastCount0 := getRatingsV1RequestCount(t, ft.east.oc)
		westCount0 := getRatingsV1RequestCount(t, ft.west.oc)
		for i := 0; i < 10; i++ {
			curl.Request(t, eastMeshBookinfoURL, nil)
		}
		numberOfRequestsEast := getRatingsV1RequestCount(t, ft.east.oc) - eastCount0
		numberOfRequestsWest := getRatingsV1RequestCount(t, ft.west.oc) - westCount0

		if numberOfRequestsEast > 0 {
			t.LogSuccessf("ratings-v1 in east-mesh received %d requests", numberOfRequestsEast)
		} else {
			t.Error("ratings-v1 in east-mesh received no requests")
		}

		if numberOfRequestsWest == 0 {
			t.LogSuccess("ratings-v1 in west-mesh received no requests")
		} else {
			t.Error("ratings-v1 in west-mesh received %d requests, but should have received none", numberOfRequestsWest)
		}
	})
}

func assertRatingsInWestMeshReceivesRequest(t TestHelper, eastMeshBookinfoURL string, ft federationTest) {
	retry.UntilSuccess(t, func(t TestHelper) {
		westCount0 := getRatingsV1RequestCount(t, ft.west.oc)
		for i := 0; i < 10; i++ {
			curl.Request(t, eastMeshBookinfoURL, nil)
		}
		numberOfRequestsWest := getRatingsV1RequestCount(t, ft.west.oc) - westCount0

		if numberOfRequestsWest > 0 {
			t.LogSuccessf("ratings-v1 in west-mesh received %d requests", numberOfRequestsWest)
		} else {
			t.Error("ratings-v1 in west-mesh received no requests, but should have received 10")
		}
	})
}

func getRatingsV1RequestCount(t TestHelper, oc *oc.OC) int {
	metrics := istio.GetProxyMetrics(t, oc,
		pod.MatchingSelector("app=ratings,version=v1", "bookinfo-ha"),
		"istio_requests_total",
		"destination_workload=ratings-v1", "reporter=destination")

	count := 0
	for _, m := range metrics {
		count += int(math.Round(*m.Counter.Value))
	}
	return count
}

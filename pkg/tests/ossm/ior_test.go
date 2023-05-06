package ossm

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

type RouteMetadata struct {
	Uid             string            `json:"uid"`
	Name            string            `json:"name"`
	ResourceVersion string            `json:"resourceVersion"`
	Labels          map[string]string `json:"labels"`
	Annotations     map[string]string `json:"annotations"`
}

type Route struct {
	Metadata RouteMetadata `json:"metadata"`
}

const (
	originalHostAnnotation = "maistra.io/original-host"
)

// TestIOR tests IOR error regarding routes recreated: https://issues.redhat.com/browse/OSSM-1974. IOR will be deprecated on 2.4 and willl be removed on 3.0
func TestIOR(t *testing.T) {
	test.NewTest(t).Groups(test.Full).Run(func(t test.TestHelper) {
		t.Log("This test verifies the behavior of IOR.")

		meshNamespace := env.GetDefaultMeshNamespace()
		meshName := env.GetDefaultSMCPName()

		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})

		host := "www.test.ocp"

		createSimpleGateway := func(t test.TestHelper) {
			t.Logf("Creating Gateway for %s host", host)
			oc.ApplyString(t, "", generateGateway("gw", meshNamespace, host))
		}

		checkSimpleGateway := func(t test.TestHelper) {
			t.Logf("Checking whether a Route is generated for %s", host)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				routes := getRoutes(t, meshNamespace)
				found := routes[0].Metadata.Annotations[originalHostAnnotation]
				if found != host {
					t.Fatalf("Expect a route set for %s host, but got %s instead", host, found)
				} else {
					t.LogSuccessf("Got an expected Route for %s", host)
				}
			})
		}

		DeployControlPlane(t)

		t.NewSubTest("check IOR off by default v2.4").Run(func(t test.TestHelper) {
			t.LogStep("Check whether the IOR has the correct default setting")
			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_4) {
				if getIORSetting(t, meshNamespace, meshName) != "false" {
					t.Fatal("Expect to find IOR disabled by default in v2.4+, but it is currently enabled")
				} else {
					t.LogSuccess("Got the expected false for IOR setting")
				}
			}
		})

		t.NewSubTest("check IOR basic functionalities").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_4) {
					removeIORCustomSetting(t, meshNamespace, meshName)
				}
			})

			t.LogStep("Ensure the IOR enabled")
			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_4) {
				enableIOR(t, meshNamespace, meshName)
			}

			t.LogStep("Check whether the IOR creates Routes for hosts specified in the Gateway")
			createSimpleGateway(t)
			checkSimpleGateway(t)
		})

		t.NewSubTest("check routes that are not deleted during v2.3 to v2.4 upgrade").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				oc.RecreateNamespace(t, meshNamespace)
				setupDefaultSMCP(t, meshNamespace)
			})

			t.LogStepf("Delete and recreate namespace %s", meshNamespace)
			oc.RecreateNamespace(t, meshNamespace)

			t.LogStep("Deploy SMCP v2.3")
			setupV23SMCP(t, meshNamespace, meshName)

			t.LogStep("Check whether IOR creates Routes for hosts specified in the Gateway")
			createSimpleGateway(t)
			checkSimpleGateway(t)

			t.LogStepf("Record the Route before the upgrade")
			before := getRoutes(t, meshNamespace)

			t.LogStep("Upgrade SMCP to v2.4")
			updateToV24SMCP(t, meshNamespace, meshName)

			t.LogStep("Check the Routes existed afther the upgrade")
			checkSimpleGateway(t)
			after := getRoutes(t, meshNamespace)

			t.LogStep("Check the Routes unchanged afther the upgrade")
			if before[0].Metadata.ResourceVersion != after[0].Metadata.ResourceVersion {
				t.Fatal("Expect the route to be unchanged, but it is changed after the upgrade")
			} else {
				t.LogSuccess("Got the same resourceVersion before and after the upgrade")
			}
		})

		t.NewSubTest("check IOR does not delete routes after deleting Istio pod").Run(func(t test.TestHelper) {
			total := 3
			nsNames := []string{}
			gateways := []string{}

			t.Cleanup(func() {
				if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_4) {
					removeIORCustomSetting(t, meshNamespace, meshName)
				}

				oc.DeleteNamespace(t, nsNames...)
				oc.ApplyString(t, meshNamespace, GetSMMRTemplate())
				oc.WaitSMMRReady(t, meshNamespace)
			})

			t.LogStep("Ensure the IOR enabled")
			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_4) {
				enableIOR(t, meshNamespace, meshName)
			}

			t.LogStepf("Create %d Gateways and they are in their own Namespace", total)
			for i := 0; i < total; i++ {
				ns := fmt.Sprintf("ns-%d", i)
				nsNames = append(nsNames, ns)
				gateways = append(gateways, generateGateway(fmt.Sprintf("gw-%d", i), ns, fmt.Sprintf("www-%d.test.ocp", i)))
			}
			oc.CreateNamespace(t, nsNames...)
			oc.ApplyString(t, "", gateways...)

			t.LogStepf("Update SMMR to include %d Namespaces", total)
			oc.ApplyString(t, meshNamespace, fmt.Sprintf(`
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  members:
  - bookinfo
  - foo
  - bar
  - legacy
  - %s
  `, strings.Join(nsNames, "\n  - ")))

			retry.UntilSuccess(t, func(t test.TestHelper) {
				routes := getRoutes(t, meshNamespace)
				if len(routes) == total {
					t.LogSuccessf("Found all %d Routes", total)
				} else {
					t.Fatalf("Expect to find %d Routes but found %d instead", total, len(routes))
				}
			})

			before := getRoutes(t, meshNamespace)
			detectRouteChanges := func() {
				retry.UntilSuccess(t, func(t test.TestHelper) {
					after := getRoutes(t, meshNamespace)

					if len(after) != total {
						t.Fatalf("Expect %d Routes, but got %d instead", total, len(after))
					}

					if fmt.Sprint(buildRouteMap(before)) != fmt.Sprint(buildRouteMap(after)) {
						t.Fatalf("Expect %d Routes remain unchanged, but they changed\nBefore: %v\nAfter: %v", total, before, after)
					}

					t.LogSuccessf("Got %d Routes unchanged", total)
				})
			}

			t.LogStepf("Check whether the Routes changes when the istio pod restarts")
			restartPod(t, meshName, meshNamespace, 10)
			detectRouteChanges()

			t.LogStepf("Check weather the Routes changes when adding new IngressGateway")
			addAdditionalIngressGateway(t, meshName, meshNamespace, "additional-test-ior-ingress-gateway")
			detectRouteChanges()
		})
	})
}

func addAdditionalIngressGateway(t test.TestHelper, meshName, meshNamespace, gatewayName string) {
	oc.Patch(t, meshNamespace,
		"smcp", meshName,
		"merge", fmt.Sprintf(`
{
  "spec": {
    "gateways": {
      "additionalIngress": {
        "%s": {}
      }
    }
  }
}
`, gatewayName))
	oc.WaitSMCPReady(t, meshNamespace, meshName)
}

func restartPod(t test.TestHelper, name, ns string, count int) {
	for i := 0; i < count; i++ {
		istiodPod := pod.MatchingSelector("app=istiod", meshNamespace)
		// t.Logf("Deleting %s pod in %s", name, ns)
		oc.DeletePod(t, istiodPod)
		oc.WaitPodRunning(t, istiodPod)
		oc.WaitPodReady(t, istiodPod)
	}
}

func buildRouteMap(routes []Route) map[string]string {
	routeMap := make(map[string]string)

	for _, route := range routes {
		routeMap[route.Metadata.Uid] = route.Metadata.ResourceVersion
	}

	return routeMap
}

func getRoutes(t test.TestHelper, ns string) []Route {
	res := shell.Executef(t, "oc -n %s get --selector 'maistra.io/generated-by=ior' --output 'jsonpath={.items}' route", ns)
	var routes []Route
	err := json.Unmarshal([]byte(res), &routes)
	if err != nil {
		t.Fatalf("Error parsing data %s: %v", res, err)
	}

	return routes
}

func setupDefaultSMCP(t test.TestHelper, ns string) {
	InstallSMCP(t, ns)
	oc.WaitSMCPReady(t, ns, env.GetDefaultSMCPName())
}

func setupV23SMCP(t test.TestHelper, ns, name string) {
	InstallSMCPVersion(t, ns, version.SMCP_2_3)
	oc.WaitSMCPReady(t, ns, name)

	oc.ApplyString(t, ns, GetSMMRTemplate())
	oc.WaitSMMRReady(t, ns)
}

func updateToV24SMCP(t test.TestHelper, ns, name string) {
	oc.Patch(t, ns,
		"smcp", name,
		"json", `[{"op": "add", "path": "/spec/version", "value": "v2.4"}]`)
	oc.WaitSMCPReady(t, ns, name)
}

func getIORSetting(t test.TestHelper, ns, name string) string {
	return shell.Executef(t,
		`oc -n %s get smcp/%s -o jsonpath='{.status.appliedValues.istio.gateways.istio-ingressgateway.ior_enabled}'`,
		ns, name)
}

func enableIOR(t test.TestHelper, ns, name string) {
	oc.Patch(t,
		ns, "smcp", name, "json",
		`[{"op": "add", "path": "/spec/gateways", "value": {"openshiftRoute": {"enabled": true}}}]`,
	)
}

func removeIORCustomSetting(t test.TestHelper, ns, name string) {
	oc.Patch(t,
		ns, "smcp", name, "json",
		`[{"op": "remove", "path": "/spec/gateways"}]`,
	)
}

func generateGateway(name, ns, host string) string {
	return fmt.Sprintf(`
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: "%s"
  namespace: "%s"
spec:
  selector:
    istio: "ingressgateway"
  servers:
  - hosts:
    - "%s"
    port:
      name: "http"
      number: 80
      protocol: "HTTP"
---`,
		name, ns, host,
	)
}

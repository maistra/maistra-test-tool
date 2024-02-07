package ossm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/maistra/maistra-test-tool/pkg/util"
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
	test.NewTest(t).Groups(test.Full, test.ARM).Run(func(t test.TestHelper) {
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

		deleteSimpleGateway := func(t test.TestHelper) {
			t.Logf("Deleting Gateway for %s host", host)
			oc.DeleteFromString(t, "", generateGateway("gw", meshNamespace, host))
		}

		checkSimpleGateway := func(t test.TestHelper) {
			t.Logf("Checking whether a Route is generated for %s", host)
			retry.UntilSuccess(t, func(t test.TestHelper) {
				routes := getRoutes(t, meshNamespace)
				if len(routes) != 1 {
					t.Fatalf("Expect a single route set for %s host, but got %s instead", host, len(routes))
				}
				found := routes[0].Metadata.Annotations[originalHostAnnotation]
				if found != host {
					t.Fatalf("Expect a route set for %s host, but got %s instead", host, found)
				} else {
					t.LogSuccessf("Got an expected Route for %s", host)
				}
			})
		}

		DeployControlPlane(t)

		t.NewSubTest("check IOR off by default v2.5").Run(func(t test.TestHelper) {
			if env.GetSMCPVersion().LessThan(version.SMCP_2_5) {
				t.Skip("Skipping until 2.5")
			}
			t.LogStep("Check whether the IOR has the correct default setting")
			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
				if getIORSetting(t, meshNamespace, meshName) != "true" {
					t.Fatal("Expect to find IOR disabled by default in v2.4+, but it is currently enabled")
				} else {
					t.LogSuccess("Got the expected true for IOR setting")
				}
			} else {
				t.Skip("IOR is not set by default versions less than v2.3")
			}
		})

		t.NewSubTest("check IOR basic functionalities").Run(func(t test.TestHelper) {
			t.Cleanup(func() {
				if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
					removeIORCustomSetting(t, meshNamespace, meshName)
				}
				deleteSimpleGateway(t)
			})

			t.LogStep("Ensure the IOR enabled")
			if env.GetSMCPVersion().GreaterThanOrEqual(version.SMCP_2_5) {
				enableIOR(t, meshNamespace, meshName)
			}

			t.LogStep("Check whether the IOR creates Routes for hosts specified in the Gateway")
			createSimpleGateway(t)
			checkSimpleGateway(t)
		})

		t.NewSubTest("check routes aren't deleted during v2.3 to v2.4 upgrade").Run(func(t test.TestHelper) {
			if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
				t.Skip("This test only applies for v2.3 to v2.4 upgrade")
			}

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
			gatewayMap := make(map[string]string)
			for i := 0; i < total; i++ {
				ns := fmt.Sprintf("ns-%d", i)
				nsNames = append(nsNames, ns)
				gatewayName := fmt.Sprintf("gw-%d", i)
				gatewayHost := fmt.Sprintf("www-%d.test.ocp", i)
				gateways = append(gateways, generateGateway(gatewayName, ns, gatewayHost))
				gatewayMap[gatewayName] = gatewayHost
			}
			oc.CreateNamespace(t, nsNames...)
			oc.ApplyString(t, "", gateways...)

			t.LogStepf("Update SMMR to include %d Namespaces", total)
			oc.ApplyString(t, meshNamespace, AppendDefaultSMMR(nsNames...))
			oc.WaitSMMRReady(t, meshNamespace)

			retry.UntilSuccess(t, func(t test.TestHelper) {
				routes := getRoutes(t, meshNamespace)
				if len(routes) != total {
					t.Fatalf("Expect to find %d Routes but found %d instead", total, len(routes))
				}

				newGatewayMap := make(map[string]string)
				for _, r := range routes {
					newGatewayMap[r.Metadata.Labels["maistra.io/gateway-name"]] = r.Metadata.Annotations[originalHostAnnotation]
				}

				var (
					err        error
					beforeYaml []byte
					afterYaml  []byte
				)
				if beforeYaml, err = yaml.Marshal(gatewayMap); err != nil {
					t.Fatalf("Failed to marshal %s", gatewayMap)
				}
				if afterYaml, err = yaml.Marshal(gatewayMap); err != nil {
					t.Fatalf("Failed to marshal %s", newGatewayMap)
				}
				if err := util.Compare(beforeYaml, afterYaml); err != nil {
					t.Fatalf("Expected %d Routes created for each Gateway but got %s.", total, err)
				}

				t.LogSuccessf("Found all %d Routes", total)
			})

			before := buildManagedRouteYamlDocument(t, meshNamespace)
			detectRouteChanges := func(t test.TestHelper) {
				after := buildManagedRouteYamlDocument(t, meshNamespace)
				if err := util.Compare(
					[]byte(before),
					[]byte(after)); err != nil {
					t.Fatalf("Expect %d Routes remain unchanged, but they changed\n%s", total, err)
				}

				t.LogSuccessf("Got %d Routes unchanged", total)
			}

			t.LogStepf("Check whether the Routes changes when the istio pod restarts multiple times")
			t.Log("Restart pod 10 times to make sure the Routes are not changed")
			count := 10
			for i := 0; i < count; i++ {
				istiodPod := pod.MatchingSelector("app=istiod", meshNamespace)
				oc.DeletePod(t, istiodPod)
				oc.WaitPodReady(t, istiodPod)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			detectRouteChanges(t)

			t.LogStepf("Check weather the Routes changes when adding new IngressGateway")
			addAdditionalIngressGateway(t, meshName, meshNamespace, "additional-test-ior-ingress-gateway")
			detectRouteChanges(t)
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

func getRoutes(t test.TestHelper, ns string) []Route {
	res := shell.Executef(t, "oc -n %s get --selector 'maistra.io/generated-by=ior' --output 'jsonpath={.items}' route", ns)
	var routes []Route
	err := json.Unmarshal([]byte(res), &routes)
	if err != nil {
		t.Fatalf("Error parsing data %s: %v", res, err)
	}

	return routes
}

func getRouteNames(t test.TestHelper, ns string) []string {
	return strings.Split(
		shell.Executef(t,
			"oc -n %s get --selector 'maistra.io/generated-by=ior' --output 'jsonpath={.items[*].metadata.name}' route",
			ns),
		" ")
}

func buildManagedRouteYamlDocument(t test.TestHelper, ns string) string {
	names := getRouteNames(t, ns)
	sort.Strings(names)

	doc := ""
	for _, name := range names {
		route := oc.GetYaml(t, ns, "route", name)

		lines := strings.Split(route, "\n")
		count := len(lines)
		found := false

		for i := 0; found == false && i < count; i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "resourceVersion") {
				found = true
				lines[i] = ""
			}
		}

		doc += fmt.Sprintf("%s\n---\n", strings.Join(lines, "\n"))
	}

	return doc
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

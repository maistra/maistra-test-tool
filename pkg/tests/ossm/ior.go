package ossm

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

type RouteMetadata struct {
	Name         string `json:"name"`
	CreationTime string `json:"creationTimestamp"`
}

type Routes struct {
	Metadata RouteMetadata `json:"metadata"`
}

func cleanupMultipleIOR() {
	util.Log.Info("Delete the namespaces and smmr")
	util.Shell(`../scripts/smmr/clean_members_50.sh`)
	//Remove the gateways
	util.Shell(`oc patch smcp/%s -n %s --type json -p='[{"op": "remove", "path": "/spec/gateways/additionalEgress"}]'`, smcpName, meshNamespace)
	util.Shell(`oc patch smcp/%s -n %s --type json -p='[{"op": "remove", "path": "/spec/gateways/additionalIngress"}]'`, smcpName, meshNamespace)
	time.Sleep(time.Duration(40) * time.Second)
}

// TestIOR tests IOR error regarding routes recreated: https://issues.redhat.com/browse/OSSM-1974. IOR will be deprecated on 2.4 and willl be removed on 3.0
func TestIOR(t *testing.T) {
	defer cleanupMultipleIOR()
	//For 2.4 we need to enable IOR, by default should be disable
	defer util.RecoverPanic(t)
	smcpVersion, _ := util.ShellMuteOutput(`oc get smcp/%s -n %s -o jsonpath='{.spec.version}'`, smcpName, meshNamespace)
	util.Log.Info("SMCP version: ", smcpVersion)
	iorEnabled, _ := util.ShellMuteOutput(`oc get smcp/%s -n %s -o jsonpath='{.status.appliedValues.istio.gateways.istio-ingressgateway.ior_enabled}'`, smcpName, meshNamespace)
	util.Log.Info("IOR enabled: ", iorEnabled)

	// Check if IOR is enabled or not by default. Note: for >= 2.4 IOR is disabled by default, so we need to enable it in another subtest. For < 2.4 IOR is enabled by default.
	t.Run("check_default_ior_state", func(t *testing.T) {
		defer util.RecoverPanic(t)
		if strings.Contains(smcpVersion, "v2.4") {
			util.Log.Info("IOR should be disabled by default")
			if iorEnabled == "true" {
				t.Errorf("IOR should be disabled by default")
			}
		} else {
			if iorEnabled == "false" {
				t.Errorf("IOR should be enabled by default")
			}
		}
	})
	// Check if we can enable ior for >= 2.4. IOR is disabled by default.
	t.Run("enable_ior", func(t *testing.T) {
		defer util.RecoverPanic(t)
		if strings.Contains(smcpVersion, "v2.4") {
			util.Log.Info("IOR should be disabled by default")
			if iorEnabled == "true" {
				t.Errorf("IOR should be disabled by default")
			} else {
				//Need to enable ior on the smcp
				util.Log.Info("Enabling IOR")
				util.Shell(`oc patch smcp/%s -n %s --type json -p='[{"op": "add", "path": "/spec/gateways/openshiftRoute/enabled", "value": true}]'`, smcpName, meshNamespace)
				_, err := util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName)
				if err != nil {
					t.Fatal("SMCP is not ready")
				}
				iorEnabled, _ := util.ShellMuteOutput(`oc get smcp/%s -n %s -o jsonpath='{.status.appliedValues.istio.gateways.istio-ingressgateway.ior_enabled}'`, smcpName, meshNamespace)
				util.Log.Info("IOR enabled: ", iorEnabled)
				if iorEnabled == "false" {
					t.Errorf("IOR should be enabled")
				}
			}
		} else {
			util.Log.Info("IOR should be enabled by default")
			if iorEnabled == "false" {
				t.Errorf("IOR should be enabled by default")
			} else {
				t.Skip("IOR is enabled by default, no need to enable it. Skipping this test")
			}
		}
	})
	util.Log.Info("Setup IOR")
	util.Log.Info("Create 50 new namespaces")
	util.Shell(`../scripts/smmr/create_members_50.sh`)
	util.Log.Info("Namespaces and smmr created...")
	util.Log.Info("Create gateway in each namespace")
	util.Shell(`../scripts/gateway/create_multiple_gateway.sh`)
	util.Log.Info("Gateways created...")
	routes, _ := util.ShellMuteOutput(`oc get -n %s route -o jsonpath='{.items}'`, meshNamespace)
	routesData := ParseRoutes(routes)
	for k, v := range routesData {
		util.Log.Info("Route: ", k, " CreationTime: ", v)
	}
	// // Check that the routes are not recreated after deleting the istiod pod
	t.Run("check_routes_recreation", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete istiod pod multiple times")
		if _, err := util.Shell(`for n in $(seq 1 10):; do oc rollout restart deployment/istiod-basic -n %s; oc -n %s wait --for condition=Ready smcp/%s --timeout 60s; done`, meshNamespace, meshNamespace, smcpName); err != nil {
			t.Fatal("SMCP is not ready after istiod pod deletion", err)
		}
		//Get the routes again to compare the routes
		routes, _ = util.ShellMuteOutput(`oc get -n %s route -o jsonpath='{.items}'`, meshNamespace)
		routesDataNew := ParseRoutes(routes)
		for k, v := range routesDataNew {
			util.Log.Info("Route: ", k, " CreationTime: ", v)
		}
		if len(routesData) != len(routesDataNew) {
			t.Errorf("The number of routes has changed")
		}
		//Compare the routes list to check that the routes are not recreated. The routes are recreated if the creation time is different
		for k, v := range routesData {
			if v != routesDataNew[k] {
				t.Errorf("The route %s has been recreated", k)
			}
		}
		//Patch the SMCP to create a aditional ingressgateway and egressgateway. Can not do it 100 time because we run out of memory on the nodes
		err := AddAdditionalGateway(10)
		if err != nil {
			t.Fatal("Error adding aditional ingress and egress", err)
		}
		routes, _ = util.ShellMuteOutput(`oc get -n %s route -o jsonpath='{.items}'`, meshNamespace)
		routesDataNew = ParseRoutes(routes)
		for k, v := range routesDataNew {
			util.Log.Info("Route: ", k, " CreationTime: ", v)
		}
		if len(routesData) != len(routesDataNew) {
			t.Errorf("The number of routes has changed")
		}
		for k, v := range routesData {
			if v != routesDataNew[k] {
				t.Errorf("The route %s has been recreated", k)
			}
		}
	})
}

func ParseRoutes(routes string) map[string]string {
	var routesdata []Routes
	routesMap := make(map[string]string)
	err := json.Unmarshal([]byte(routes), &routesdata)
	if err != nil {
		util.Log.Error("Error parsing routes: ", err)
		return nil
	}
	for _, route := range routesdata {
		if strings.Contains(route.Metadata.Name, "gw-http") && route.Metadata.Name != "" {
			routesMap[route.Metadata.Name] = route.Metadata.CreationTime
		}
	}
	return routesMap
}

func AddAdditionalGateway(number int) error {
	//Patch the SMCP to create a aditionals (by number variable) ingressgateway and egressgateway
	var err error
	for i := 0; i < number; i++ {
		_, err = util.Shell(`oc patch smcp/%s -n %s --type merge -p '{"spec":{"gateways":{"additionalEgress":
		{"eg%d":{"enabled":true,"namespace":"%s","runtime":{"container":{"resources":
		{"limits":{"cpu":"1000m","memory":"1028Mi"},"requests":{"cpu":"300m","memory":"250Mi"}}},
		"deployment":{"autoScaling":{"enabled":false},"replicas":1}}}},"additionalIngress":
		{"ig%d":{"enabled":true,"namespace":"%s","runtime":
		{"container":{"resources":{"limits":{"cpu":"1000m","memory":"1024Mi"},"requests":
		{"cpu":"200m","memory":"150Mi"}}},"deployment":{"autoScaling":{"enabled":false}}}}}}}}'`,
			smcpName, meshNamespace, i, meshNamespace, i, meshNamespace)
		if err != nil {
			return err
		}
		util.Log.Info("Verify SMCP status and pods")
		if _, err = util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName); err != nil {
			return err
		}
	}
	return err

}

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
	util.Shell(`../scripts/smmr/clean_members_ior.sh`)
	//Add patch smcp to remove the gateways
	util.Shell(`oc patch smcp/%s -n %s --type merge -p '{"spec":{"gateways":{}}'`, smcpName, meshNamespace)
	time.Sleep(time.Duration(40) * time.Second)
}

// TestIOR tests IOR error regarding routes recreated: https://issues.redhat.com/browse/OSSM-1974. IOR will be deprecated on 2.4 and willl be removed on 3.0
func TestIOR(t *testing.T) {
	defer cleanupMultipleIOR()
	defer util.RecoverPanic(t)
	util.Log.Info("Setup IOR")
	util.Log.Info("Create 100 new namespaces")
	util.Shell(`../scripts/smmr/create_members_ior.sh`)
	util.Log.Info("Namespaces and smmr created...")
	util.Log.Info("Create gateway in each namespace")
	util.Shell(`../scripts/gateway/create_multiple_gateway.sh`)
	util.Log.Info("Gateways created...")
	routes, _ := util.ShellMuteOutput(`oc get -n istio-system route -o jsonpath='{.items}'`)
	routesData := ParseRoutes(routes)
	for k, v := range routesData {
		util.Log.Info("Route: ", k, " CreationTime: ", v)
	}

	// // Check that the routes are not recreated after deleting the istiod pod
	t.Run("delete_istiod_check_routes", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Delete istiod pod multiple times")
		if _, err := util.Shell(`for n in $(seq 1 10):; do oc rollout restart deployment/istiod-basic -n istio-system; oc -n %s wait --for condition=Ready smcp/%s --timeout 60s; done`, meshNamespace, smcpName); err != nil {
			t.Fatal("SMCP is not ready after istiod pod deletion", err)
		}
		//Get the routes again to compare the routes
		routes, _ = util.ShellMuteOutput(`oc get -n istio-system route -o jsonpath='{.items}'`)
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
	})
	// Check that the routes are not recreated after create a new ingressgateway and egressgateway
	t.Run("create_aditional_ingress_egress_check_routes", func(t *testing.T) {
		defer util.RecoverPanic(t)
		//Patch the SMCP to create a aditional ingressgateway and egressgateway
		err := AddAdditionalGateway(100)
		if err != nil {
			t.Fatal("Error adding aditional ingress and egress", err)
		}
		routes, _ = util.ShellMuteOutput(`oc get -n istio-system route -o jsonpath='{.items}'`)
		routesDataNew := ParseRoutes(routes)
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

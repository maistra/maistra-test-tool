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
	routesdata := ParseRoutes(routes)
	for k, v := range routesdata {
		util.Log.Info("Route: ", k, " CreationTime: ", v)
	}

	// Check that the routes are not recreated after deleting the istiod pod
	t.Run("delete_istiod_check_routes", func(t *testing.T) {
		defer util.RecoverPanic(t)

	})
	// Check that the routes are not recreated after create a new ingressgateway and egressgateway
	t.Run("create_ingress_egress_check_routes", func(t *testing.T) {
		defer util.RecoverPanic(t)

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

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
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

type config struct {
	oc                *oc.OC
	smcpName          string
	smcpNamespace     string
	bookinfoNamespace string
	region            string
	zone              string
}

type federationTest struct {
	testdataPath string

	west config
	east config

	controlPlaneInstaller func(t TestHelper, ft federationTest)
	bookinfoInstaller     func(t TestHelper, ft federationTest)
	checker               func(t TestHelper, ft federationTest)
}

func (ft federationTest) run(t TestHelper) {
	ocWest := ft.west.oc
	ocEast := ft.east.oc
	singleCluster := ocWest == ocEast

	t.Cleanup(func() {
		if singleCluster {
			// if we're using a single cluster, we can delete all namespaces at the same time (it's faster)
			ocWest.DeleteNamespace(t, ft.west.smcpNamespace, ft.east.smcpNamespace, ft.west.bookinfoNamespace, ft.east.bookinfoNamespace)
		} else {
			ocWest.DeleteNamespace(t, ft.west.smcpNamespace, ft.west.bookinfoNamespace)
			ocEast.DeleteNamespace(t, ft.east.smcpNamespace, ft.east.bookinfoNamespace)
		}
	})

	t.LogStep("Create projects for west-mesh and east-mesh")
	ocWest.CreateNamespace(t, ft.west.smcpNamespace, ft.west.bookinfoNamespace)
	ocEast.CreateNamespace(t, ft.east.smcpNamespace, ft.east.bookinfoNamespace)

	t.LogStep("Install control planes for west-mesh and east-mesh")
	ft.controlPlaneInstaller(t, ft)

	t.LogStep("Wait for west-mesh and east-mesh installation to complete")
	ocWest.WaitSMCPReady(t, ft.west.smcpNamespace, ft.west.smcpName)
	ocEast.WaitSMCPReady(t, ft.east.smcpNamespace, ft.east.smcpName)

	t.LogStep("Retrieve peer addresses and ports")
	var peer1Address, peer2Address string
	if singleCluster {
		t.Log("Using ClusterIP service for ingress")
		peer1Address = "east-mesh-ingress.west-mesh-system.svc.cluster.local"
		peer2Address = "west-mesh-ingress.east-mesh-system.svc.cluster.local"
	} else {
		t.Log("Using LoadBalancer service for ingress")
		peer1Address = getLoadBalancerIngressAddress(t, ft.west, "east-mesh-ingress")
		peer2Address = getLoadBalancerIngressAddress(t, ft.east, "west-mesh-ingress")
	}

	westMeshInfo := PeerInfo{Address: peer1Address, DiscoveryPort: "8188", ServicePort: "15443", Region: ft.west.region, Zone: ft.west.zone}
	eastMeshInfo := PeerInfo{Address: peer2Address, DiscoveryPort: "8188", ServicePort: "15443", Region: ft.east.region, Zone: ft.east.zone}
	t.Logf("west-mesh: address: %s; discovery port: %v, service port: %v", westMeshInfo.Address, westMeshInfo.DiscoveryPort, westMeshInfo.ServicePort)
	t.Logf("east-mesh: address: %s; discovery port: %v, service port: %v", eastMeshInfo.Address, eastMeshInfo.DiscoveryPort, eastMeshInfo.ServicePort)

	t.LogStep("Retrieve root certificates")
	westMeshInfo.CARootCert = getRootCertificate(t, ft.west)
	eastMeshInfo.CARootCert = getRootCertificate(t, ft.east)

	t.LogStep("Install ServiceMeshPeer and ExportedServiceSet in west-mesh")
	ocWest.ApplyTemplateFile(t, ft.west.smcpNamespace, ft.testdataPath+"/west-mesh/configmap.yaml", eastMeshInfo)
	ocWest.ApplyTemplateFile(t, ft.west.smcpNamespace, ft.testdataPath+"/west-mesh/servicemeshpeer.yaml", eastMeshInfo)
	ocWest.ApplyFile(t, ft.west.smcpNamespace, ft.testdataPath+"/west-mesh/exportedserviceset.yaml")

	t.LogStep("Install ServiceMeshPeer and ImportedServiceSet in east-mesh")
	ocEast.ApplyTemplateFile(t, ft.east.smcpNamespace, ft.testdataPath+"/east-mesh/configmap.yaml", westMeshInfo)
	ocEast.ApplyTemplateFile(t, ft.east.smcpNamespace, ft.testdataPath+"/east-mesh/servicemeshpeer.yaml", westMeshInfo)
	ocEast.ApplyTemplateFile(t, ft.east.smcpNamespace, ft.testdataPath+"/east-mesh/importedserviceset.yaml", westMeshInfo)

	ft.bookinfoInstaller(t, ft)

	t.LogStep("Wait for all bookinfo pods in west-mesh and east-mesh to be ready")
	ocWest.WaitAllPodsReady(t, ft.west.bookinfoNamespace)
	ocEast.WaitAllPodsReady(t, ft.east.bookinfoNamespace)

	t.LogStep("Check if west-mesh and east-mesh are connected to each other")
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t TestHelper) { // this typically takes 5 minutes on AWS
		ocWest.Invoke(t,
			`oc -n west-mesh-system get servicemeshpeer east-mesh -o json`,
			assert.OutputContains(`"connected": true`, // TODO: must also check for lastSyncTime, since the peer might be connected, but not synced
				"west-mesh is connected to east-mesh",
				"west-mesh is not connected to east-mesh"))
		ocEast.Invoke(t,
			`oc -n east-mesh-system get servicemeshpeer west-mesh -o json`,
			assert.OutputContains(`"connected": true`, // TODO: must also check for lastSyncTime, since the peer might be connected, but not synced
				"east-mesh is connected to west-mesh",
				"east-mesh is not connected to west-mesh"))
	})

	ft.checker(t, ft)
}

func getLoadBalancerIngressAddress(t TestHelper, c config, serviceName string) string {
	var address string
	retryFor10Minutes := retry.Options().MaxAttempts(6 * 10).DelayBetweenAttempts(10 * time.Second)
	retry.UntilSuccessWithOptions(t, retryFor10Minutes, func(t TestHelper) {
		// try to get the load balancer ip
		address = c.oc.Invokef(t, `oc -n %s get svc %s -o jsonpath="{.status.loadBalancer.ingress[].ip}"`, c.smcpNamespace, serviceName)
		if address != "" {
			return
		}

		// try to get the load balancer hostname
		address = c.oc.Invokef(t, `oc -n %s get svc %s -o jsonpath="{.status.loadBalancer.ingress[].hostname}"`, c.smcpNamespace, serviceName)
		if address != "" {
			return
		}

		t.Fatalf("could not get ingress address from LoadBalancer service %s/%s", c.smcpNamespace, serviceName)
	})
	return address
}

func getRootCertificate(t TestHelper, c config) string {
	configMap := "istio-ca-root-cert"
	key := "root-cert.pem"

	t.Logf("Get key %s from ConfigMap %s", key, configMap)
	data := c.oc.GetConfigMapData(t, c.smcpNamespace, configMap)
	return data[key]
}

func installSMCPandSMMR(t TestHelper, c config, smcpFile, smmrFile string) {
	t.Logf("Install ServiceMeshControlPlane %s in namespace %s", c.smcpName, c.smcpNamespace)
	c.oc.ApplyTemplateFile(t, c.smcpNamespace, smcpFile, map[string]string{
		"Version": env.GetSMCPVersion().String(),
	})

	t.Log("Create ServiceMeshMemberRoll")
	c.oc.ApplyFile(t, c.smcpNamespace, smmrFile)
}

type PeerInfo struct {
	CARootCert    string
	Address       string
	DiscoveryPort string
	ServicePort   string
	Region        string
	Zone          string
}

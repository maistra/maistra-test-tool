// Copyright 2026 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package migration

import (
	_ "embed"
	"fmt"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/check/require"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var (
	//go:embed testdata/federation/east/smcp.yaml
	eastSMCPTmpl string

	//go:embed testdata/federation/west/smcp.yaml
	westSMCPTmpl string

	//go:embed testdata/federation/east/servicemeshpeer.yaml
	eastServiceMeshPeerTmpl string

	//go:embed testdata/federation/west/servicemeshpeer.yaml
	westServiceMeshPeerTmpl string

	//go:embed testdata/federation/east/importedserviceset.yaml
	eastImportedServiceSetTmpl string

	//go:embed testdata/federation/west/exportedserviceset.yaml
	westExportedServiceSet string

	//go:embed testdata/federation/east/istio.yaml
	eastIstioTmpl string

	//go:embed testdata/federation/west/istio.yaml
	westIstioTmpl string

	//go:embed testdata/federation/west/eastwestgateway.yaml
	westEastWestGateway string

	//go:embed testdata/federation/west/expose-services-gateway.yaml
	westExposeServicesGateway string

	//go:embed testdata/federation/west/serviceentry-httpbin-a.yaml
	westServiceEntryHttpbinA string

	//go:embed testdata/federation/east/remote-gateway.yaml
	eastRemoteGatewayTmpl string

	//go:embed testdata/federation/east/serviceentry-httpbin-a.yaml
	eastServiceEntryHttpbinA string

	//go:embed testdata/federation/east/serviceentry-httpbin-b.yaml
	eastServiceEntryHttpbinB string
)

const (
	eastMeshNamespace = "istio-system"
	westMeshNamespace = "istio-system"
	clientNamespace   = "client"
	httpbinANamespace = "a"
	httpbinBNamespace = "b"
	istioIngressNs    = "istio-ingress"
)

var federationTestdataPath = env.GetRootDir() + "/pkg/tests/tasks/migration/testdata/federation"

type smcpConfig struct {
	oc          *oc.OC
	namespace   string
	name        string
	trustDomain string
}

type peerInfo struct {
	Address          string
	CARootCert       string
	PeerTrustDomain  string
	PeerEgressSA     string
	PeerMeshName     string
	PeerIngressName  string
	LocalEgressName  string
	LocalIngressName string
	LocalMeshName    string
}

// TestFederationMigration tests migration from OSSM 2.6 federation to OSSM 3.0 multi-cluster.
// This test follows the migration guide from:
// https://github.com/openshift-service-mesh/sail-operator/tree/main/docs/ossm/ossm2-migration/federation
func TestFederationMigration(t *testing.T) {
	test.NewTest(t).MinVersion(version.SMCP_2_6).Groups(test.Migration).Run(func(t test.TestHelper) {
		kubeconfig2 := env.GetKubeconfig2()
		if kubeconfig2 == "" {
			t.Skip("This test requires KUBECONFIG2 environment variable pointing to the second cluster")
		}

		ocEast := oc.DefaultOC
		ocWest := oc.WithKubeconfig(kubeconfig2)

		eastConfig := smcpConfig{
			oc:          ocEast,
			namespace:   eastMeshNamespace,
			name:        "basic",
			trustDomain: "east.local",
		}
		westConfig := smcpConfig{
			oc:          ocWest,
			namespace:   westMeshNamespace,
			name:        "basic",
			trustDomain: "west.local",
		}

		t.Cleanup(func() {
			t.LogStep("Cleaning up resources")
			ocEast.DeleteResource(t, "", "Istio", "default")
			ocWest.DeleteResource(t, "", "Istio", "default")
			ocEast.DeleteResource(t, "", "IstioCNI", "default")
			ocWest.DeleteResource(t, "", "IstioCNI", "default")
			ocEast.DeleteNamespace(t, eastMeshNamespace, clientNamespace)
			ocWest.DeleteNamespace(t, westMeshNamespace, httpbinANamespace, httpbinBNamespace, istioIngressNs)
		})

		t.LogStep("Create namespaces")
		ocEast.CreateNamespace(t, eastMeshNamespace, clientNamespace)
		ocWest.CreateNamespace(t, westMeshNamespace, httpbinANamespace, httpbinBNamespace)

		t.LogStep("Create cacerts secrets for custom CA")
		createCACertsSecrets(t, ocEast, eastMeshNamespace, federationTestdataPath+"/east")
		createCACertsSecrets(t, ocWest, westMeshNamespace, federationTestdataPath+"/west")

		t.LogStep("Deploy SMCP 2.6 with federation gateways in both clusters")
		smcpVersion := env.GetSMCPVersion().String()
		ocEast.ApplyTemplateString(t, eastMeshNamespace, eastSMCPTmpl, map[string]string{"Version": smcpVersion})
		ocWest.ApplyTemplateString(t, westMeshNamespace, westSMCPTmpl, map[string]string{"Version": smcpVersion})

		t.LogStep("Wait for SMCPs to be ready")
		ocEast.WaitSMCPReady(t, eastMeshNamespace, eastConfig.name)
		ocWest.WaitSMCPReady(t, westMeshNamespace, westConfig.name)

		t.LogStep("Configure mesh federation")
		configureFederation(t, eastConfig, westConfig)

		t.LogStep("Deploy applications")
		deployFederationApplications(t, ocEast, ocWest)

		t.LogStep("Export and import services")
		configureServiceExportImport(t, eastConfig, westConfig)

		t.LogStep("Verify federation connectivity before migration")
		verifyFederationConnectivity(t, ocEast, clientNamespace)

		t.LogStep("Configure SMCP for migration - add trustDomainAliases and network settings")
		configureSMCPForMigration(t, eastConfig, westConfig)

		t.LogStep("Deploy OSSM 3.0 control planes")
		deployOSSM3ControlPlanes(t, ocEast, ocWest)

		t.LogStep("Deploy east-west gateway and expose services in west cluster")
		deployEastWestGateway(t, ocWest)

		t.LogStep("Create ServiceEntry and WorkloadEntry for imported services")
		configureMultiClusterResources(t, ocEast, ocWest)

		t.LogStep("Verify connectivity via new gateway (before migrating proxies)")
		ocEast.Invoke(t, fmt.Sprintf("oc rollout restart deployment/curl -n %s", clientNamespace))
		ocEast.WaitDeploymentRolloutComplete(t, clientNamespace, "curl")
		verifyFederationConnectivity(t, ocEast, clientNamespace)

		t.LogStep("Remove federation resources")
		removeFederationResources(t, eastConfig, westConfig)

		t.LogStep("Migrate workloads to OSSM 3.0 control plane")
		migrateWorkloadsToOSSM3(t, eastConfig, westConfig)

		t.LogStep("Verify connectivity after migration")
		verifyFederationConnectivity(t, ocEast, clientNamespace)
	})
}

func createCACertsSecrets(t test.TestHelper, oc *oc.OC, namespace, certDir string) {
	oc.CreateGenericSecretFromFiles(t, namespace, "cacerts",
		fmt.Sprintf("root-cert.pem=%s/root-cert.pem", certDir),
		fmt.Sprintf("ca-cert.pem=%s/ca-cert.pem", certDir),
		fmt.Sprintf("ca-key.pem=%s/ca-key.pem", certDir),
		fmt.Sprintf("cert-chain.pem=%s/cert-chain.pem", certDir),
	)
}

func configureFederation(t test.TestHelper, east, west smcpConfig) {
	// Get ingress addresses
	var eastIngressAddr, westIngressAddr string
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t test.TestHelper) {
		eastIngressAddr = east.oc.GetLoadBalancerAddress(t, east.namespace, "federation-ingress")
		if eastIngressAddr == "" {
			t.Error("East ingress address not available yet")
		}
	})

	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t test.TestHelper) {
		westIngressAddr = west.oc.GetLoadBalancerAddress(t, west.namespace, "federation-ingress")
		if westIngressAddr == "" {
			t.Error("West ingress address not available yet")
		}
	})

	t.Logf("East ingress address: %s", eastIngressAddr)
	t.Logf("West ingress address: %s", westIngressAddr)

	// Create CA root cert ConfigMaps in the peer clusters
	west.oc.CreateConfigMapFromFiles(t, west.namespace, "east-mesh-ca-root-cert",
		federationTestdataPath+"/east/root-cert.pem")
	east.oc.CreateConfigMapFromFiles(t, east.namespace, "west-mesh-ca-root-cert",
		federationTestdataPath+"/west/root-cert.pem")

	// Configure ServiceMeshPeer in west cluster (to connect to east)
	westPeerInfo := peerInfo{
		Address:          eastIngressAddr,
		PeerTrustDomain:  "east.local",
		PeerEgressSA:     "east.local/ns/istio-system/sa/federation-egress-service-account",
		PeerMeshName:     "east-mesh",
		LocalIngressName: "federation-ingress",
		LocalEgressName:  "federation-egress",
	}
	west.oc.ApplyTemplateString(t, west.namespace, westServiceMeshPeerTmpl, westPeerInfo)

	// Configure ServiceMeshPeer in east cluster (to connect to west)
	eastPeerInfo := peerInfo{
		Address:          westIngressAddr,
		PeerTrustDomain:  "west.local",
		PeerEgressSA:     "west.local/ns/istio-system/sa/federation-egress-service-account",
		PeerMeshName:     "west-mesh",
		LocalIngressName: "federation-ingress",
		LocalEgressName:  "federation-egress",
	}
	east.oc.ApplyTemplateString(t, east.namespace, eastServiceMeshPeerTmpl, eastPeerInfo)

	// Wait for peers to connect
	t.LogStep("Wait for ServiceMeshPeers to connect")
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t test.TestHelper) {
		east.oc.GetYaml(t, east.namespace, "servicemeshpeer", "west-mesh",
			assert.OutputContains("connected: true", "east-mesh connected to west-mesh", "east-mesh not connected to west-mesh"))
		west.oc.GetYaml(t, west.namespace, "servicemeshpeer", "east-mesh",
			assert.OutputContains("connected: true", "west-mesh connected to east-mesh", "west-mesh not connected to east-mesh"))
	})
}

func getRootCertFromConfigMap(t test.TestHelper, oc *oc.OC, namespace string) string {
	return oc.GetConfigMapData(t, namespace, "istio-ca-root-cert")["root-cert.pem"]
}

func deployFederationApplications(t test.TestHelper, ocEast, ocWest *oc.OC) {
	sidecarPatch := `{"spec":{"template":{"metadata":{"annotations":{"sidecar.istio.io/inject":"true"}}}}}`

	// Deploy curl client in east cluster
	ocEast.Label(t, "", "Namespace", clientNamespace, "istio-injection=enabled")
	ocEast.ApplyFile(t, clientNamespace, "https://raw.githubusercontent.com/istio/istio/master/samples/curl/curl.yaml")
	ocEast.Patch(t, clientNamespace, "deploy", "curl", "merge", sidecarPatch)
	ocEast.WaitDeploymentRolloutComplete(t, clientNamespace, "curl")

	// Deploy httpbin in west cluster (namespaces a and b)
	ocWest.Label(t, "", "Namespace", httpbinANamespace, "istio-injection=enabled")
	ocWest.ApplyFile(t, httpbinANamespace, "https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml")
	ocWest.Patch(t, httpbinANamespace, "deploy", "httpbin", "merge", sidecarPatch)
	ocWest.WaitDeploymentRolloutComplete(t, httpbinANamespace, "httpbin")

	ocWest.Label(t, "", "Namespace", httpbinBNamespace, "istio-injection=enabled")
	ocWest.ApplyFile(t, httpbinBNamespace, "https://raw.githubusercontent.com/istio/istio/master/samples/httpbin/httpbin.yaml")
	ocWest.Patch(t, httpbinBNamespace, "deploy", "httpbin", "merge", sidecarPatch)
	ocWest.WaitDeploymentRolloutComplete(t, httpbinBNamespace, "httpbin")
}

func configureServiceExportImport(t test.TestHelper, east, west smcpConfig) {
	// Export services from west to east
	west.oc.ApplyString(t, west.namespace, westExportedServiceSet)

	// Import services in east from west
	east.oc.ApplyString(t, east.namespace, eastImportedServiceSetTmpl)

	// Wait for exported services to appear
	t.Log("Wait for exported services to be available")
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t test.TestHelper) {
		west.oc.GetYaml(t, west.namespace, "exportedserviceset", "east-mesh",
			assert.OutputContains("httpbin.a.svc.east-mesh-exports.local", "httpbin.a exported", "httpbin.a not exported"))
		west.oc.GetYaml(t, west.namespace, "exportedserviceset", "east-mesh",
			assert.OutputContains("httpbin.b.svc.east-mesh-exports.local", "httpbin.b exported", "httpbin.b not exported"))
	})

	// Wait for imported services to appear
	t.Log("Wait for imported services to be available")
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t test.TestHelper) {
		east.oc.GetYaml(t, east.namespace, "importedserviceset", "west-mesh",
			assert.OutputContains("httpbin.a.svc.west-mesh-imports.local", "httpbin.a imported", "httpbin.a not imported"))
		east.oc.GetYaml(t, east.namespace, "importedserviceset", "west-mesh",
			assert.OutputContains("httpbin.b.svc.cluster.local", "httpbin.b imported as local", "httpbin.b not imported"))
	})
}

func verifyFederationConnectivity(t test.TestHelper, ocEast *oc.OC, clientNs string) {
	retry.UntilSuccess(t, func(t test.TestHelper) {
		// Test connectivity to httpbin.a via west-mesh-imports
		cmd := fmt.Sprintf("oc exec -n %s deploy/curl -c curl -- curl -s httpbin.a.svc.west-mesh-imports.local:8000/headers", clientNs)
		ocEast.Invoke(t, cmd, require.OutputContains("Host", "Request to httpbin.a succeeded", "Request to httpbin.a failed"))

		// Test connectivity to httpbin.b via cluster.local (importAsLocal: true)
		cmd = fmt.Sprintf("oc exec -n %s deploy/curl -c curl -- curl -s httpbin.b.svc.cluster.local:8000/headers", clientNs)
		ocEast.Invoke(t, cmd, require.OutputContains("Host", "Request to httpbin.b succeeded", "Request to httpbin.b failed"))
	})
}

func configureSMCPForMigration(t test.TestHelper, east, west smcpConfig) {
	generatePatch := func(trustDomainAlias, networkName string) string {
		return fmt.Sprintf(`{
	"spec": {
		"techPreview": {
			"meshConfig": {
				"trustDomainAliases": ["%s"]
			}
		},
		"runtime": {
			"components": {
				"pilot": {
					"container": {
						"env": {
							"PILOT_MULTI_NETWORK_DISCOVER_GATEWAY_API": "true"
						}
					}
				}
			}
		},
		"cluster": {
			"network": "%s"
		},
		"security": {
			"manageNetworkPolicy": false
		}
	}
}`, trustDomainAlias, networkName)
	}

	east.oc.Patch(t, east.namespace, "smcp", east.name, "merge", generatePatch("west.local", "network-east-mesh"))
	west.oc.Patch(t, west.namespace, "smcp", west.name, "merge", generatePatch("east.local", "network-west-mesh"))

	east.oc.WaitSMCPReady(t, east.namespace, east.name)
	west.oc.WaitSMCPReady(t, west.namespace, west.name)
}

func deployEastWestGateway(t test.TestHelper, ocWest *oc.OC) {
	ocWest.CreateNamespace(t, istioIngressNs)
	ocWest.ApplyString(t, istioIngressNs, westEastWestGateway)
	ocWest.ApplyString(t, istioIngressNs, westExposeServicesGateway)
	ocWest.WaitDeploymentRolloutComplete(t, istioIngressNs, "eastwestgateway-istio")

	// Create ServiceEntry for httpbin.a with west-mesh-imports hostname
	ocWest.ApplyString(t, httpbinANamespace, westServiceEntryHttpbinA)
}

func configureMultiClusterResources(t test.TestHelper, ocEast, ocWest *oc.OC) {
	// Get E/W gateway address from west cluster
	var westEWGWAddr string
	retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60).DelayBetweenAttempts(10*time.Second), func(t test.TestHelper) {
		westEWGWAddr = ocWest.GetLoadBalancerAddress(t, istioIngressNs, "eastwestgateway-istio")
		if westEWGWAddr == "" {
			t.Error("West E/W gateway address not available yet")
		}
	})
	t.Logf("West E/W gateway address: %s", westEWGWAddr)

	// Create remote gateway in east cluster
	ocEast.ApplyTemplateString(t, eastMeshNamespace, eastRemoteGatewayTmpl, map[string]string{
		"Address": westEWGWAddr,
	})

	// Create ServiceEntry and WorkloadEntry for imported services
	ocEast.ApplyString(t, eastMeshNamespace, eastServiceEntryHttpbinA)
	ocEast.ApplyString(t, eastMeshNamespace, eastServiceEntryHttpbinB)
}

func deployOSSM3ControlPlanes(t test.TestHelper, ocEast, ocWest *oc.OC) {
	// Get west root cert for east Istio config
	westRootCert := getRootCertFromConfigMap(t, ocWest, westMeshNamespace)
	eastRootCert := getRootCertFromConfigMap(t, ocEast, eastMeshNamespace)

	// Deploy IstioCNI in both clusters
	istioCNI := `apiVersion: sailoperator.io/v1
kind: IstioCNI
metadata:
  name: default
spec:
  version: v1.24-latest
  namespace: istio-cni`

	ocEast.CreateNamespace(t, "istio-cni")
	ocWest.CreateNamespace(t, "istio-cni")
	ocEast.ApplyString(t, "", istioCNI)
	ocWest.ApplyString(t, "", istioCNI)

	// Wait for IstioCNI to be ready
	ocEast.WaitFor(t, "", "IstioCNI", "default", "condition=Ready")
	ocWest.WaitFor(t, "", "IstioCNI", "default", "condition=Ready")

	// Deploy Istio in east cluster
	ocEast.ApplyTemplateString(t, "", eastIstioTmpl, map[string]string{
		"WestRootCert": westRootCert,
	})

	// Deploy Istio in west cluster
	ocWest.ApplyTemplateString(t, "", westIstioTmpl, map[string]string{
		"EastRootCert": eastRootCert,
	})

	// Wait for Istio to be ready
	ocEast.WaitFor(t, "", "Istio", "default", "condition=Ready")
	ocWest.WaitFor(t, "", "Istio", "default", "condition=Ready")
}

func removeFederationResources(t test.TestHelper, east, west smcpConfig) {
	// Remove ImportedServiceSet and ExportedServiceSet
	east.oc.DeleteResource(t, east.namespace, "importedserviceset", "west-mesh")
	west.oc.DeleteResource(t, west.namespace, "exportedserviceset", "east-mesh")

	// Remove ServiceMeshPeers
	east.oc.DeleteResource(t, east.namespace, "servicemeshpeer", "west-mesh")
	west.oc.DeleteResource(t, west.namespace, "servicemeshpeer", "east-mesh")

	// Remove federation gateways from SMCPs
	gatewaysPatch := `[{"op": "remove", "path": "/spec/gateways/additionalIngress"}, {"op": "remove", "path": "/spec/gateways/additionalEgress"}]`
	east.oc.Patch(t, east.namespace, "smcp", east.name, "json", gatewaysPatch)
	west.oc.Patch(t, west.namespace, "smcp", west.name, "json", gatewaysPatch)
}

func migrateWorkloadsToOSSM3(t test.TestHelper, east, west smcpConfig) {
	// Get OSSM 3 revision names
	eastRevName := east.oc.GetJson(t, "", "Istio", "default", "{.status.activeRevisionName}")
	westRevName := west.oc.GetJson(t, "", "Istio", "default", "{.status.activeRevisionName}")

	t.Logf("East revision: %s, West revision: %s", eastRevName, westRevName)

	// Migrate client namespace in east cluster
	east.oc.Label(t, "", "Namespace", clientNamespace,
		fmt.Sprintf("%s istio-injection- istio.io/rev=%s", maistraIgnoreLabel, eastRevName))
	east.oc.Invoke(t, fmt.Sprintf("oc rollout restart deployment/curl -n %s", clientNamespace))
	east.oc.WaitDeploymentRolloutComplete(t, clientNamespace, "curl")

	// Verify curl pod has new revision
	retry.UntilSuccess(t, func(t test.TestHelper) {
		annotations := oc.GetPodAnnotations(t, pod.MatchingSelector("app=curl", clientNamespace))
		if actual := annotations["istio.io/rev"]; actual != eastRevName {
			t.Errorf("Expected revision %s, got %s", eastRevName, actual)
		}
	})

	// Migrate httpbin namespaces in west cluster
	west.oc.Label(t, "", "Namespace", httpbinANamespace,
		fmt.Sprintf("%s istio-injection- istio.io/rev=%s", maistraIgnoreLabel, westRevName))
	west.oc.Invoke(t, fmt.Sprintf("oc rollout restart deployment/httpbin -n %s", httpbinANamespace))
	west.oc.WaitDeploymentRolloutComplete(t, httpbinANamespace, "httpbin")

	west.oc.Label(t, "", "Namespace", httpbinBNamespace,
		fmt.Sprintf("%s istio-injection- istio.io/rev=%s", maistraIgnoreLabel, westRevName))
	west.oc.Invoke(t, fmt.Sprintf("oc rollout restart deployment/httpbin -n %s", httpbinBNamespace))
	west.oc.WaitDeploymentRolloutComplete(t, httpbinBNamespace, "httpbin")

	// Migrate east-west gateway
	west.oc.Label(t, "", "gateways.gateway.networking.k8s.io", "eastwestgateway",
		fmt.Sprintf("-n %s istio.io/rev=%s maistra.io/ignore=true", istioIngressNs, westRevName))

	// Wait for gateway to be redeployed
	west.oc.WaitDeploymentRolloutComplete(t, istioIngressNs, "eastwestgateway-istio")

	// Verify httpbin pods have new revision
	retry.UntilSuccess(t, func(t test.TestHelper) {
		annotationsA := west.oc.GetPodAnnotations(t, pod.MatchingSelector("app=httpbin", httpbinANamespace))
		if actual := annotationsA["istio.io/rev"]; actual != westRevName {
			t.Errorf("httpbin.a: Expected revision %s, got %s", westRevName, actual)
		}

		annotationsB := west.oc.GetPodAnnotations(t, pod.MatchingSelector("app=httpbin", httpbinBNamespace))
		if actual := annotationsB["istio.io/rev"]; actual != westRevName {
			t.Errorf("httpbin.b: Expected revision %s, got %s", westRevName, actual)
		}
	})
}

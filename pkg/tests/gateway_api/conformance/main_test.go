package conformace

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/gateway-api/conformance/tests"
	"sigs.k8s.io/gateway-api/conformance/utils/suite"
	v1 "sigs.k8s.io/gateway-api/apis/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/gatewayapi"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var (
	meshNamespace = env.GetDefaultMeshNamespace()
	gwInfraNamespace = "gateway-conformance-infra"
)

func TestMain(m *testing.M) {
	test.NewSuite(m).
		Setup(ossm.BasicSetup).
		Run()
}

func TestGatewayApiConformance(t *testing.T) {
	NewTest(t).Id("T41").Groups(Full, InterOp, ARM).Run(func(t TestHelper) {
		if env.GetSMCPVersion().LessThan(version.SMCP_2_6) {
			t.Skip("Gateway API Conformance tests only work on 2.6")
		}

		smcpName := env.GetDefaultSMCPName()

		ossm.InstallSMCPCustom(t, meshNamespace, ossm.DefaultSMCP())

		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `
spec:
  addons:
    grafana:
      enabled: false
  techPreview:
    gatewayAPI:
      enabled: true
`)

		oc.ApplyString(t, meshNamespace, smmr)
		oc.WaitSMCPReady(t, meshNamespace, ossm.DefaultSMCP().Name)
		oc.WaitSMMRReady(t, meshNamespace)

		// create the namespace infra and deploy a network policy into it
		oc.CreateNamespace(t, gwInfraNamespace)
		oc.ApplyString(t, gwInfraNamespace, networkPolicy)

		t.LogStep("Install Gateway API CRD's")
		gatewayapi.InstallSupportedVersion(t, env.GetSMCPVersion())

		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})


		cfg, err := config.GetConfig()
		if err != nil {
			t.Fatalf("Error loading Kubernetes config: %v", err)
		}
		client, err := client.New(cfg, client.Options{})
		if err != nil {
			t.Fatalf("Error initializing Kubernetes client: %v", err)
		}
		clientset, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			t.Fatalf("Error initializing Kubernetes REST client: %v", err)
		}

		v1.Install(client.Scheme())

		supportedFeatures := sets.New[suite.SupportedFeature]().
			Insert(suite.GatewayExtendedFeatures.UnsortedList()...).
			Insert(suite.HTTPRouteCoreFeatures.UnsortedList()...).
			//Insert(suite.ReferenceGrantCoreFeatures.UnsortedList()...).
			Insert(suite.HTTPRouteExtendedFeatures.UnsortedList()...)

		opts := suite.Options{
			Client:  client,
			Clientset: clientset,
			GatewayClassName:			"istio",
			NamespaceLabels:            map[string]string{"istio-injection": "enabled"},
			SupportedFeatures:          supportedFeatures,
			CleanupBaseResources:		false, // FIXME
			//RunTest:                    "HTTPExactPathMatching",
		}

		t.LogStep("Running Gateway API conformance test suite")
		csuite := suite.New(opts)
		csuite.Setup(t.T())
		csuite.Run(t.T(), tests.ConformanceTests)

	})
}

const smmr = `
apiVersion: maistra.io/v1
kind: ServiceMeshMemberRoll
metadata:
  name: default
spec:
  memberSelectors:
  - matchLabels:
      istio-injection: enabled`

const networkPolicy = `
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: gatewayingress
  namespace: gateway-conformance-infra
spec:
  podSelector:
    matchLabels:
      maistra-control-plane: istio-system
  ingress:
    - {}
  policyTypes:
  - Ingress
`

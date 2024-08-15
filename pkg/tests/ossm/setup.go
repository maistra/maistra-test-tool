package ossm

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/ns"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

type SMCP struct {
	Name      string
	Namespace string
	Version   version.Version
	Rosa      bool

	ClusterWideProxy bool
	HttpProxy        string
	HttpsProxy       string
	NoProxy          string
}

// WithName returns a copy of this SMCP with the name changed to the specified name
func (s SMCP) WithName(name string) SMCP {
	s.Name = name // NOTE: this doesn't change the SMCP that the WithName() method was invoked on, because the method receiver is not a pointer
	return s
}

func (s SMCP) WithVersion(ver version.Version) SMCP {
	s.Version = ver
	return s
}

var (
	//go:embed yaml/smcp.yaml
	smcpTemplate string

	//go:embed yaml/smmr.yaml
	smmr string

	//go:embed yaml/subscription-jaeger.yaml
	jaegerSubscription string

	//go:embed yaml/subscription-kiali.yaml
	kialiSubscription string

	//go:embed yaml/subscription-ossm.yaml
	ossmSubscription string

	smcpName      = env.GetDefaultSMCPName()
	meshNamespace = env.GetDefaultMeshNamespace()
	rootDir       = env.GetRootDir()
	profileFile   = rootDir + "/pkg/tests/ossm/yaml/profiles/"
)

func DefaultSMCP() SMCP {
	return SMCP{
		Name:      env.GetDefaultSMCPName(),
		Namespace: env.GetDefaultMeshNamespace(),
		Version:   env.GetSMCPVersion(),
		Rosa:      env.IsRosa(),
	}
}

// Install nightly build operators from quay.io. This is used in Jenkins daily build pipeline.
func installNightlyOperators(t test.TestHelper) {
	ns := env.GetOperatorNamespace()
	oc.ApplyString(t, ns, jaegerSubscription)
	oc.ApplyString(t, ns, kialiSubscription)
	oc.ApplyString(t, ns, ossmSubscription)
	oc.WaitDeploymentRolloutComplete(t, ns, "istio-operator", "jaeger-operator", "kiali-operator")
}

func BasicSetup(t test.TestHelper) {
	t.T().Helper()

	if env.IsNightly() {
		installNightlyOperators(t)
	}
	oc.CreateNamespace(t, meshNamespace, ns.Bookinfo, ns.Foo, ns.Bar, ns.Legacy, ns.MeshExternal)
}

func DeployControlPlane(t test.TestHelper) {
	t.T().Helper()
	t.LogStep("Apply default SMCP and SMMR manifests")
	smcpValues := DefaultSMCP()
	clusterWideProxy := oc.GetProxy(t)
	if clusterWideProxy != nil {
		smcpValues.ClusterWideProxy = true
		smcpValues.HttpProxy = clusterWideProxy.HTTPProxy
		smcpValues.HttpsProxy = clusterWideProxy.HTTPSProxy
		smcpValues.NoProxy = clusterWideProxy.NoProxy
	}
	InstallSMCPCustom(t, meshNamespace, smcpValues)
	oc.ApplyString(t, meshNamespace, smmr)
	oc.WaitSMCPReady(t, meshNamespace, DefaultSMCP().Name)
	oc.WaitSMMRReady(t, meshNamespace)
}

func InstallSMCP(t test.TestHelper, ns string) {
	InstallSMCPVersion(t, ns, env.GetSMCPVersion())
}

func InstallSMCPVersion(t test.TestHelper, ns string, ver version.Version) {
	InstallSMCPCustom(t, ns, DefaultSMCP().WithVersion(ver))
}

func InstallSMCPCustom(t test.TestHelper, ns string, smcp SMCP) {
	oc.ApplyString(t, ns, getSMCPManifestCustom(t, smcp))
}

func DeleteSMCPVersion(t test.TestHelper, ns string, ver version.Version) {
	DeleteSMCPCustom(t, ns, DefaultSMCP().WithVersion(ver))
}

func DeleteSMCPCustom(t test.TestHelper, ns string, smcp SMCP) {
	oc.DeleteFromString(t, ns, getSMCPManifestCustom(t, smcp))
}

func getSMCPManifestCustom(t test.TestHelper, smcp SMCP) string {
	return template.Run(t, smcpTemplate, smcp)
}

func GetSMMRTemplate() string {
	return smmr
}

func AppendDefaultSMMR(namespaces ...string) string {
	return fmt.Sprintf(`
%s
  - %s
  `, smmr, strings.Join(namespaces, "\n  - "))
}

func GetProfileFile() string {
	return profileFile
}

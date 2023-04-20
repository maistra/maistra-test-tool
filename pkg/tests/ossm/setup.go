package ossm

import (
	_ "embed"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
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

func SetupEnvVars(t test.TestHelper) {
	env.InitEnvVarsFromFile()
}

func BasicSetup(t test.TestHelper) {
	t.T().Helper()
	SetupEnvVars(t)

	if env.Getenv("NIGHTLY", "false") == "true" {
		installNightlyOperators(t)
	}
	oc.CreateNamespace(t, meshNamespace, "bookinfo", "foo", "bar", "legacy", "mesh-external")
}

func DeployControlPlane(t test.TestHelper) {
	t.T().Helper()
	t.LogStep("Apply default SMCP and SMMR manifests")
	InstallSMCP(t, meshNamespace)
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

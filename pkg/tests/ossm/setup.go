package ossm

import (
	_ "embed"
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type SMCP struct {
	Name      string `default:"basic"`
	Namespace string `default:"istio-system"`
	Rosa      bool   `default:"false"`
}

var (
	//go:embed yaml/subscription-jaeger.yaml
	jaegerSubscription string

	//go:embed yaml/subscription-kiali.yaml
	kialiSubscription string

	//go:embed yaml/subscription-ossm.yaml
	ossmSubscription string
)

var (
	smcpName      = env.GetDefaultSMCPName()
	meshNamespace = env.GetDefaultMeshNamespace()
	Smcp          = template.SMCP{
		Name:      smcpName,
		Namespace: meshNamespace,
		Rosa:      env.IsRosa()}
	ipv6 = env.Getenv("IPV6", "false")
)

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
	if ipv6 == "true" {
		t.Log("Running the test with IPv6 configuration")
	}

	if env.Getenv("NIGHTLY", "false") == "true" {
		installNightlyOperators(t)
	}
	oc.CreateNamespace(t, meshNamespace, "bookinfo", "foo", "bar", "legacy", "mesh-external")
}

func DeployControlPlane(t test.TestHelper) {
	t.T().Helper()
	t.LogStep("Apply default SMCP and SMMR manifests")
	tmpl := GetSMCPTemplate(env.GetDefaultSMCPVersion())
	oc.ApplyTemplate(t, meshNamespace, tmpl, Smcp)
	oc.ApplyString(t, meshNamespace, smmr)
	oc.WaitSMCPReady(t, meshNamespace, Smcp.Name)
	oc.WaitSMMRReady(t, meshNamespace)
}

func GetDefaultSMCPTemplate() string {
	return GetSMCPTemplate(env.GetDefaultSMCPVersion())
}

func GetSMCPTemplate(version string) string {
	versionTemplates := GetSMCPTemplates()

	if tmpl, ok := versionTemplates[version]; ok {
		return tmpl
	} else {
		panic(fmt.Sprintf("Unsupported SMCP version: %s", version))
	}
}

func InstallSMCP(t test.TestHelper, ns, version string) {
	oc.ApplyTemplate(t, ns, GetSMCPTemplate(version), Smcp)
}

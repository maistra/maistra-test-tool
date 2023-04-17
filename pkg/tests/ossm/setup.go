package ossm

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
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

func createNamespaces() {
	log.Log.Info("creating namespaces")
	util.ShellSilent(`oc new-project bookinfo`)
	util.ShellSilent(`oc new-project foo`)
	util.ShellSilent(`oc new-project bar`)
	util.ShellSilent(`oc new-project legacy`)
	util.ShellSilent(`oc new-project mesh-external`)
}

// Install nightly build operators from quay.io. This is used in Jenkins daily build pipeline.
func installNightlyOperators() {
	util.KubeApplyContents(env.GetOperatorNamespace(), jaegerSubscription)
	util.KubeApplyContents(env.GetOperatorNamespace(), kialiSubscription)
	util.KubeApplyContents(env.GetOperatorNamespace(), ossmSubscription)
	time.Sleep(time.Duration(60) * time.Second)
	util.CheckPodRunning(env.GetOperatorNamespace(), "name=istio-operator")
	time.Sleep(time.Duration(30) * time.Second)
}

func SetupEnvVars() {
	env.InitEnvVarsFromFile()
}

func BasicSetup() {
	SetupEnvVars()

	log.Log.Info("Starting Basic Setup")
	createNamespaces()
	if env.Getenv("NIGHTLY", "false") == "true" {
		installNightlyOperators()
	}
	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
}

// Initialize a default SMCP and SMMR
func SetupNamespacesAndControlPlane() {
	BasicSetup()
	tmpl := GetSMCPTemplate(env.GetDefaultSMCPVersion())
	util.KubeApplyContents(meshNamespace, util.RunTemplate(tmpl, Smcp))
	util.KubeApplyContents(meshNamespace, smmr)
	util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, Smcp.Name)
	util.Shell(`oc -n %s wait --for condition=Ready smmr/default --timeout 180s`, meshNamespace)
	if ipv6 == "true" {
		log.Log.Info("Running the test with IPv6 configuration")
	}
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

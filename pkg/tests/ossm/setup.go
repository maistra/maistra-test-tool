package ossm

import (
	_ "embed"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

type SMCP struct {
	Name      string `default:"basic"`
	Namespace string `default:"istio-system"`
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
	smcpName      = env.Getenv("SMCPNAME", "basic")
	meshNamespace = env.Getenv("MESHNAMESPACE", "istio-system")
	smcp          = SMCP{smcpName, meshNamespace}
	ipv6          = env.Getenv("IPV6", "false")
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
	util.KubeApplyContents("openshift-operators", jaegerSubscription)
	util.KubeApplyContents("openshift-operators", kialiSubscription)
	util.KubeApplyContents("openshift-operators", ossmSubscription)
	time.Sleep(time.Duration(60) * time.Second)
	util.CheckPodRunning("openshift-operators", "name=istio-operator")
	time.Sleep(time.Duration(30) * time.Second)
}

func BasicSetup() {
	log.Log.Info("Setting up namespaces and OSSM control plane")
	createNamespaces()
	if env.Getenv("NIGHTLY", "false") == "true" {
		installNightlyOperators()
	}
	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
}

// Initialize a default SMCP and SMMR
func SetupNamespacesAndControlPlane() {
	BasicSetup()
	versionTemplates := GetSMCPTemplates()
	smcpVersion := env.GetDefaultSMCPVersion()
	template, ok := versionTemplates[smcpVersion]
	if !ok {
		log.Log.Errorf("Unsupported SMCP version: %s", smcpVersion)
		return
	}
	util.KubeApplyContents(meshNamespace, util.RunTemplate(template, smcp))
	util.KubeApplyContents(meshNamespace, smmr)
	util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcp.Name)
	util.Shell(`oc -n %s wait --for condition=Ready smmr/default --timeout 180s`, meshNamespace)
	if ipv6 == "true" {
		log.Log.Info("Running the test with IPv6 configuration")
	}
}

// Initialize a default SMCP and SMMR
func SetupOnlyNamespaces() {
	BasicSetup()
	// TODO: add more setup steps for test who do not need SMCP. If you need to add more steps, please add them here. If not we can remove this function.
}

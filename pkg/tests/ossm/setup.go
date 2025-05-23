// Copyright 2024 Red Hat, Inc.
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

type Istio struct {
	Name      string
	Namespace string

	// Template will override the default template.
	Template string
}

type SMCP struct {
	Name          string
	Namespace     string
	Version       version.Version
	Rosa          bool
	ClusterWideCp bool
	TracingType   string

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

func DefaultSMCP(t test.TestHelper) SMCP {
	return SMCP{
		Name:          env.GetDefaultSMCPName(),
		Namespace:     env.GetDefaultMeshNamespace(),
		Version:       env.GetSMCPVersion(),
		Rosa:          env.IsRosa(),
		ClusterWideCp: false,
		TracingType:   getDefaultTracingType(t),
	}
}

func DefaultClusterWideSMCP(t test.TestHelper) SMCP {
	return SMCP{
		Name:          env.GetDefaultSMCPName(),
		Namespace:     env.GetDefaultMeshNamespace(),
		Version:       env.GetSMCPVersion(),
		Rosa:          env.IsRosa(),
		ClusterWideCp: true,
		TracingType:   getDefaultTracingType(t),
	}
}

func DefaultIstio() Istio {
	return Istio{
		Name:      env.GetIstioName(),
		Namespace: env.GetIstioNamespace(),
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

func DeployControlPlane(t test.TestHelper) SMCP {
	t.T().Helper()
	t.LogStep("Apply default SMCP and SMMR manifests")
	smcpValues := DefaultSMCP(t)
	clusterWideProxy := oc.GetProxy(t)
	if clusterWideProxy != nil {
		smcpValues.ClusterWideProxy = true
		smcpValues.HttpProxy = clusterWideProxy.HTTPProxy
		smcpValues.HttpsProxy = clusterWideProxy.HTTPSProxy
		smcpValues.NoProxy = clusterWideProxy.NoProxy
	}
	InstallSMCPCustom(t, meshNamespace, smcpValues)
	oc.ApplyString(t, meshNamespace, smmr)
	oc.WaitSMCPReady(t, meshNamespace, smcpValues.Name)
	oc.WaitSMMRReady(t, meshNamespace)
	return smcpValues
}

func DeployClusterWideControlPlane(t test.TestHelper) {
	t.T().Helper()
	t.LogStep("Apply ClusterWide SMCP")
	smcp := DefaultClusterWideSMCP(t)
	InstallSMCPCustom(t, meshNamespace, smcp)
	oc.WaitSMCPReady(t, meshNamespace, smcp.Name)
}

func InstallSMCP(t test.TestHelper, ns string) {
	InstallSMCPVersion(t, ns, env.GetSMCPVersion())
}

func InstallSMCPVersion(t test.TestHelper, ns string, ver version.Version) {
	InstallSMCPCustom(t, ns, DefaultSMCP(t).WithVersion(ver))
}

func InstallSMCPCustom(t test.TestHelper, ns string, smcp SMCP) {
	oc.ApplyString(t, ns, getSMCPManifestCustom(t, smcp))
}

func DeleteSMCPVersion(t test.TestHelper, ns string, ver version.Version) {
	DeleteSMCPCustom(t, ns, DefaultSMCP(t).WithVersion(ver))
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

func getDefaultTracingType(t test.TestHelper) string {
	// jaeger is not available on SMCP 2.6 or OCP 4.19+, so use Jaeger tracing only for SMCP 2.5 and lower and OCP 4.18 and lower
	if env.GetSMCPVersion().LessThanOrEqual(version.SMCP_2_5) && version.ParseVersion(oc.GetOCPVersion(t)).LessThanOrEqual(version.OCP_4_18) {
		return "Jaeger"
	} else {
		return "None"
	}
}

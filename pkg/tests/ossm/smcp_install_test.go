// Copyright 2021 Red Hat, Inc.
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

package ossm

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/template"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

type vars struct {
	Name      string
	Namespace string
}

func installDefaultSMCP(t TestHelper, ns string) {
	version := env.Getenv("SMCPVERSION", "2.4")
	t.Log("Tear down test case, install default SMCP version: %s", version)
	// TODO: add the logic to install specific version from env variable SMCP version
	oc.RecreateNamespace(t, ns)
	vars := vars{
		Name:      smcp.Name,
		Namespace: smcp.Namespace,
	}
	oc.ApplyString(t, meshNamespace, template.Run(t, smcpV24_template, vars))
	oc.ApplyString(t, meshNamespace, smmr)
	if env.IsRosa() {
		oc.PatchWithMerge(
			t, meshNamespace,
			fmt.Sprintf("smcp/%s", smcpName),
			`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
	}
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
	oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
	t.LogStep("Check SMCP is Ready")
}

func TestSMCPInstall(t *testing.T) {
	NewTest(t).Id("A1").Groups(Smoke, Full, InterOp, ARM).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		t.Cleanup(func() {
			installDefaultSMCP(t, meshNamespace)
		})
		vars := vars{
			Name:      smcp.Name,
			Namespace: smcp.Namespace,
		}

		t.NewSubTest("smcp_test_install_2.4").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.4")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV24_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})

		t.NewSubTest("smcp_test_uninstall_2.4").Run(func(t TestHelper) {
			t.LogStep("Delete SMCP v2.4 in namespace " + meshNamespace)
			oc.DeleteFromString(t, meshNamespace, smmr)
			oc.DeleteFromString(t, meshNamespace, template.Run(t, smcpV24_template, vars))
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.AllResourcesDeleted(t,
					meshNamespace,
					assert.OutputContains("No resources found in",
						"All resources deleted from namespace",
						"Still waiting for resources to be deleted from namespace"))
			})
		})

		t.NewSubTest("smcp_test_install_2.3").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.3")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV23_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})

		t.NewSubTest("smcp_test_uninstall_2.3").Run(func(t TestHelper) {
			t.LogStep("Delete SMCP v2.3 in namespace " + meshNamespace)
			oc.DeleteFromString(t, meshNamespace, smmr)
			oc.DeleteFromString(t, meshNamespace, template.Run(t, smcpV23_template, vars))
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.AllResourcesDeleted(t,
					meshNamespace,
					assert.OutputContains("No resources found in",
						"All resources deleted from namespace",
						"Still waiting for resources to be deleted from namespace"))
			})
		})

		t.NewSubTest("smcp_test_install_2.2").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.2")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV22_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})

		t.NewSubTest("smcp_test_uninstall_2.2").Run(func(t TestHelper) {
			t.LogStep("Delete SMCP v2.2 in namespace " + meshNamespace)
			oc.DeleteFromString(t, meshNamespace, smmr)
			oc.DeleteFromString(t, meshNamespace, template.Run(t, smcpV22_template, vars))
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.AllResourcesDeleted(t,
					meshNamespace,
					assert.OutputContains("No resources found in",
						"All resources deleted from namespace",
						"Still waiting for resources to be deleted from namespace"))
			})
		})

		t.NewSubTest("smcp_test_install_2.1").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.1")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV21_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})

		t.NewSubTest("smcp_test_uninstall_2.1").Run(func(t TestHelper) {
			t.LogStep("Delete SMCP v2.1 in namespace " + meshNamespace)
			oc.DeleteFromString(t, meshNamespace, smmr)
			oc.DeleteFromString(t, meshNamespace, template.Run(t, smcpV21_template, vars))
			retry.UntilSuccess(t, func(t TestHelper) {
				oc.AllResourcesDeleted(t,
					meshNamespace,
					assert.OutputContains("No resources found in",
						"All resources deleted from namespace",
						"Still waiting for resources to be deleted from namespace"))
			})
		})

		t.NewSubTest("smcp_test_upgrade_2.1_to_2.2").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.1")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV21_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
			t.LogStep("Upgrade SMCP from v2.1 to v2.2")
			oc.ApplyTemplate(t, meshNamespace, smcpV22_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})

		t.NewSubTest("smcp_test_upgrade_2.2_to_2.3").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.2")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV22_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
			t.LogStep("Upgrade SMCP from v2.2 to v2.3")
			oc.ApplyTemplate(t, meshNamespace, smcpV23_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})

		t.NewSubTest("smcp_test_upgrade_2.3_to_2.4").Run(func(t TestHelper) {
			t.LogStep("Crete Namespace and Install SMCP v2.3")
			oc.CreateNamespace(t, meshNamespace)
			oc.ApplyTemplate(t, meshNamespace, smcpV22_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
			t.LogStep("Upgrade SMCP from v2.3 to v2.4")
			oc.ApplyTemplate(t, meshNamespace, smcpV24_template, vars)
			if env.IsRosa() {
				oc.PatchWithMerge(
					t, meshNamespace,
					fmt.Sprintf("smcp/%s", smcpName),
					`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
			}
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
			oc.ApplyString(t, meshNamespace, smmr)
			t.LogStep("Check SMCP is Ready")
			oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
		})
	})
}

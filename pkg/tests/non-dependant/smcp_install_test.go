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

package non_dependant

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	smmr             = ossm.GetSMMRTemplate()
	versionTemplates = ossm.GetSMCPTemplates()
)

func TestSMCPInstall(t *testing.T) {
	NewTest(t).Id("A1").Groups(Smoke, Full, InterOp, ARM).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		t.Cleanup(func() {
			oc.RecreateNamespace(t, meshNamespace)
		})

		versions := []string{"2.1", "2.2", "2.3", "2.4"}

		// Testing install of SMCP for all supported version
		for i := 0; i < len(versions); i++ {
			version := versions[i]
			smcpTemplate := versionTemplates[version]
			t.NewSubTest("install_" + version).Run(func(t TestHelper) {
				t.LogStep("Delete Namespace, Create Namespace and Install SMCP v" + version)
				oc.RecreateNamespace(t, meshNamespace)
				assertSMCPDeploysAndIsReady(t, smcpTemplate, smcp)
				assertSMCPUninstallComplete(t, smcpTemplate, smcp)
			})
		}

		// Testing upgrade of SMCP to all supported version
		for i := 0; i < len(versions)-1; i++ {
			fromVersion := versions[i]
			toVersion := versions[i+1]
			fromTemplate := versionTemplates[fromVersion]
			toTemplate := versionTemplates[toVersion]

			t.NewSubTest(fmt.Sprintf("upgrade_%s_to_%s", fromVersion, toVersion)).Run(func(t TestHelper) {
				oc.RecreateNamespace(t, meshNamespace)
				assertSMCPDeploysAndIsReady(t, fromTemplate, smcp)
				t.LogStep(fmt.Sprintf("Upgrade SMCP from v%s to v%s", fromVersion, toVersion))
				assertSMCPDeploysAndIsReady(t, toTemplate, smcp)
			})
		}
	})
}
func assertSMCPDeploysAndIsReady(t test.TestHelper, smcpTemplate string, data interface{}) {
	t.LogStep("Install SMCP")
	oc.ApplyTemplate(t, meshNamespace, smcpTemplate, data)
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
	oc.ApplyString(t, meshNamespace, smmr)
	t.LogStep("Check SMCP is Ready")
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
}

func assertSMCPUninstallComplete(t test.TestHelper, smcpTemplate string, data interface{}) {
	t.LogStep("Delete SMCP in namespace " + meshNamespace)
	oc.DeleteFromString(t, meshNamespace, smmr)
	oc.DeleteFromTemplate(t, meshNamespace, smcpTemplate, data)
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.GetAllResources(t,
			meshNamespace,
			assert.OutputContains("No resources found in",
				"All resources deleted from namespace",
				"Still waiting for resources to be deleted from namespace"))
	})
}

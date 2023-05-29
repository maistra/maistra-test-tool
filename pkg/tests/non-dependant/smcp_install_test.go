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
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var (
	VERSIONS = []*version.Version{
		&version.SMCP_2_0,
		&version.SMCP_2_1,
		&version.SMCP_2_2,
		&version.SMCP_2_3,
		&version.SMCP_2_4,
	}
)

func TestSMCPInstall(t *testing.T) {
	NewTest(t).Id("A1").Groups(Smoke, Full, InterOp, ARM, Disconnected).Run(func(t TestHelper) {
		t.Cleanup(func() {
			// Delete meshNamespace and recreate it in case that we have any issue with the test to avoid have a non-wanted smcp version installed
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.NewSubTest("install and verify default features").Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP %s becomes ready", env.GetSMCPVersion())

			t.LogStepf("Delete and re-create Namespace")
			oc.RecreateNamespace(t, meshNamespace)

			t.LogStepf("Create SMCP %s and verify it becomes ready", env.GetSMCPVersion())
			assertSMCPDeploysAndIsReady(t, env.GetSMCPVersion())

			t.LogStep("verify the default features for the smcp version")
			if env.GetSMCPVersion().Equals(version.SMCP_2_4) {
				t.Log("verification for default enable or disable features in SMCP 2.4: ClusterWide")

				t.LogStep("Verify ClusterWide feature is not enabled")
				oc.GetYaml(t,
					meshNamespace,
					"smcp", "basic",
					assert.OutputDoesNotContain(
						"mode: ClusterWide",
						"Clusterwide feature is disabled by default",
						"Cluster wide feature is enabled by default"))
			}

			t.LogStep("Delete SMCP and verify if this deletes all resources")
			assertUninstallDeletesAllResources(t, env.GetSMCPVersion())
		})

		toVersion := env.GetSMCPVersion()
		fromVersion := getPreviousVersion(t, toVersion)

		t.NewSubTest(fmt.Sprintf("upgrade %s to %s", fromVersion, toVersion)).Run(func(t TestHelper) {
			t.Logf("This test checks whether SMCP becomes ready after it's upgraded from %s to %s", fromVersion, toVersion)

			t.LogStepf("Delete and re-create Namespace")
			oc.RecreateNamespace(t, meshNamespace)

			t.LogStepf("Create SMCP %s and verify it becomes ready", fromVersion)
			assertSMCPDeploysAndIsReady(t, fromVersion)

			t.LogStepf("Upgrade SMCP from %s to %s", fromVersion, toVersion)
			assertSMCPDeploysAndIsReady(t, toVersion)
		})
	})
}

func assertSMCPDeploysAndIsReady(t test.TestHelper, ver version.Version) {
	t.LogStep("Install SMCP")
	ossm.InstallSMCPVersion(t, meshNamespace, ver)
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
	oc.ApplyString(t, meshNamespace, ossm.GetSMMRTemplate())
	t.LogStep("Check SMCP is Ready")
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
}

func assertUninstallDeletesAllResources(t test.TestHelper, ver version.Version) {
	t.LogStep("Delete SMCP in namespace " + meshNamespace)
	oc.DeleteFromString(t, meshNamespace, ossm.GetSMMRTemplate())
	ossm.DeleteSMCPVersion(t, meshNamespace, ver)
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.GetAllResources(t,
			meshNamespace,
			assert.OutputContains("No resources found in",
				"All resources deleted from namespace",
				"Still waiting for resources to be deleted from namespace"))
	})
}

func getPreviousVersion(t test.TestHelper, ver version.Version) version.Version {
	var prevVersion *version.Version
	for _, v := range VERSIONS {
		if *v == ver {
			if prevVersion == nil {
				t.Logf("version %s is the first supported version", ver)
			}
			return *prevVersion
		}
		prevVersion = v
	}
	panic(fmt.Sprintf("version %s not found in VERSIONS", ver))
}

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

package operator

import (
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestOperatorCanReconcileSMCPWhenIstiodOffline(t *testing.T) {
	test.NewTest(t).Groups(test.Full, test.Disconnected, test.ARM).Run(func(t test.TestHelper) {
		t.Log("This test checks if the operator can reconcile an SMCP even if the istiod pod is missing")
		t.Log("See https://issues.redhat.com/browse/OSSM-3235")

		if env.GetSMCPVersion().LessThan(version.SMCP_2_4) {
			t.Skip("Skipped because OSSM-3235 is only fixed in 2.4+")
		}

		meshNamespace := env.GetDefaultMeshNamespace()
		smcpName := env.GetDefaultSMCPName()
		istiodDeployment := fmt.Sprintf("istiod-%s", smcpName)

		t.Cleanup(func() {
			oc.ScaleDeploymentAndWait(t, meshNamespace, istiodDeployment, 1)
		})

		t.LogStep("Install SMCP and wait for it to be Ready")
		ossm.InstallSMCP(t, meshNamespace)
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Scale istiod to zero replicas, so that the validation webhook goes offline")
		oc.ScaleDeploymentAndWait(t, meshNamespace, istiodDeployment, 0)

		t.LogStep("Force SMCP to be reconciled")
		oc.TouchSMCP(t, meshNamespace, smcpName)

		t.LogStep("Wait for SMCP to be ready; if this doesn't happen, the ValidationWebhookConfiguration is probably missing the correct objectSelector")
		oc.WaitSMCPReady(t, meshNamespace, smcpName)
	})
}

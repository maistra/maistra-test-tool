// Copyright 2021 Red Hat, Inc.
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
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestTLSVersionSMCP(t *testing.T) {
	NewTest(t).Id("T26").Groups(Full, ARM, InterOp).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		t.Log("This test checks if the SMCP updated the tls.minProtocolVersion to TLSv1_0, TLSv1_1, and tls.maxProtocolVersion to TLSv1_3.")
		t.Cleanup(func() {
			oc.Patch(t, meshNamespace,
				"smcp", smcpName,
				"json",
				`[{"op": "remove", "path": "/spec/security/controlPlane/tls"}]`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.NewSubTest("minVersion_TLSv1_0").Run(func(t TestHelper) {
			t.LogStep("Update SMCP spec.security.controlPlane.tls.minProtocolVersion: TLSv1_0")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_0"}}}}}`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.NewSubTest("minVersion_TLSv1_1").Run(func(t TestHelper) {
			t.LogStep("Check to see if the SMCP minProtocolVersion is TLSv1_1")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `{"spec":{"security":{"controlPlane":{"tls":{"minProtocolVersion":"TLSv1_1"}}}}}`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})

		t.NewSubTest("maxVersion_TLSv1_3").Run(func(t TestHelper) {
			t.LogStep("Check to see if the SMCP maxProtocolVersion is TLSv1_3")
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", `{"spec":{"security":{"controlPlane":{"tls":{"maxProtocolVersion":"TLSv1_3"}}}}}`)
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})
	})
}

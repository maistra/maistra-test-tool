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
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestSMCPAddons(t *testing.T) {
	NewTest(t).Id("T34").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)

		// Created a subtest because we need to add more test related to Addons in the future.
		t.NewSubTest("3scale_addon").Run(func(t TestHelper) {
			t.LogStep("Enable 3scale in a SMCP expecting to get validation error.")
			t.Cleanup(func() {
				shell.Execute(t, fmt.Sprintf(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"addons":{"3scale":{"enabled":false}}}}' || true`, meshNamespace, smcpName))
			})
			shell.Execute(t,
				fmt.Sprintf(`oc patch -n %s smcp/%s --type merge -p '{"spec":{"addons":{"3scale":{"enabled":true}}}}' || true`, meshNamespace, smcpName),
				assert.OutputContains("support for 3scale has been removed",
					"Got expected validation error: support for 3scale has been removed",
					"The validation error was not shown as expected"))
			oc.WaitSMCPReady(t, meshNamespace, smcpName)
		})
	})
}

// Copyright Red Hat, Inc.
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
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestLogging(t *testing.T) {
	NewTest(t).Groups(Full, Disconnected, ARM, Persistent).Run(func(t TestHelper) {
		t.Log("This test verifies allowed logging levels for the control plane")
		t.Log("See https://issues.redhat.com/browse/OSSM-6331")

		t.LogStep("Deploy control plane")
		DeployControlPlane(t)

		t.Cleanup(func() {
			oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", loggingDefaut)
		})

		t.LogStep("Try to patch SMCP with unsupported log levels (trace, critical), patch should fail")
		shell.Execute(t,
			fmt.Sprintf("oc patch smcp/%s -n %s --type merge --patch '%s' || true", smcpName, meshNamespace, errorLoggingComponentLevelsSMCP),
			assert.OutputContains(`Error from server (BadRequest): admission webhook "smcp.validation.maistra.io" denied the request:`,
				"Patch failed as expected due to unsupported log levels",
				"Patch succeeded unexpectedly, unsupported log levels were not rejected"),
			assert.OutputContains(`istiod doesn't support 'trace' log level`,
				"Patch failed as expected due to unsupported log levels",
				"Patch succeeded unexpectedly, unsupported log levels were not rejected"),
			assert.OutputContains(`istiod doesn't support 'critical' log level`,
				"Patch failed as expected due to unsupported log levels",
				"Patch succeeded unexpectedly, unsupported log levels were not rejected"),
		)

		t.LogStep("Wait SMCP ready")
		oc.WaitSMCPReady(t, meshNamespace, smcpName)

		t.LogStep("Try to patch SMCP with supported log levels (none, error, warn, info, debug, fail)")
		oc.Patch(t, meshNamespace, "smcp", smcpName, "merge", loggingComponentLevelsSMCP)

		t.LogStep("Wait SMCP ready")
		oc.WaitSMCPReady(t, meshNamespace, smcpName)
	})
}

const errorLoggingComponentLevelsSMCP = `
spec:
  general:
    logging:
      componentLevels:
        ads: none
        analysis: error
        authn: warn
        ca: info 
        installer: trace #error
        resource: critical #error
`

const loggingComponentLevelsSMCP = `
spec:
  general:
    logging:
      componentLevels:
        ads: none
        analysis: error
        authn: warn
        ca: info
        installer: debug
        resource: fatal
`

const loggingDefaut = `
spec:
  general:
    logging:
      componentLevels:
        default: warn
`

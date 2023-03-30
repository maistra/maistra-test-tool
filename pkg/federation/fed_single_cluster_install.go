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

package federation

import (
	"os"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/heredoc"
	. "github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func TestSingleClusterFed(t *testing.T) {
	NewTest(t).LegacyID("T31", "T8").Groups(ARM, Full).Run(func(t TestHelper) {
		defer func() {
			Log.Info("Cleanup ...")
			shell.Execute(t, heredoc.Doc(`
				pushd ../testdata/examples/federation \
				&& export MESH1_KUBECONFIG=~/.kube/config \
				&& export MESH2_KUBECONFIG=~/.kube/config \
				&& ./cleanup.sh`))
		}()

		t.Log("Test federation install in a single cluster")
		t.Log("Reference: https://github.com/maistra/istio/blob/maistra-2.3/samples/federation/base/install.sh")

		t.Log("Running install.sh...")
		shell.Execute(t, `pushd ../testdata/examples/federation \
			&& export MESH1_KUBECONFIG=~/.kube/config \
			&& export MESH2_KUBECONFIG=~/.kube/config \
			&& ./install.sh`)

		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60), func(t TestHelper) {
			shell.Execute(t,
				`oc -n mesh1-system get servicemeshpeer mesh2 -o json`,
				assert.OutputContains(
					`"connected": true`, // TODO: must also check for lastSyncTime, since the peer might be connected, but not synced
					"mesh2 is connected in mesh1",
					"mesh2 is not connected in mesh1"))
		})

		retry.UntilSuccess(t, func(t TestHelper) {
			shell.Execute(t,
				`oc -n mesh2-system get servicemeshpeer mesh1 -o json`,
				assert.OutputContains(
					`"connected": true`, // TODO: must also check for lastSyncTime, since the peer might be connected, but not synced
					"mesh1 is connected in mesh2",
					"mesh1 is not connected in mesh2"))
		})

		retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(60), func(t TestHelper) {
			shell.Execute(t,
				`oc -n mesh2-system get importedservicesets mesh1 -o json`,
				assert.OutputContains(
					"mongodb.bookinfo.svc.mesh2-exports.local",
					"mongodb service from mesh1 found in mesh2",
					"mongodb service from mesh1 not found in mesh2"),
				assert.OutputContains(
					"ratings.bookinfo.svc.mesh2-exports.local",
					"ratings service from mesh1 found in mesh2",
					"ratings service from mesh1 not found in mesh2"))
		})
	})
}

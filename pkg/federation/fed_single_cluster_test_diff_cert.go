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
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupSingleClusterFedDiffCert() {
	util.Log.Info("Cleanup ...")
	util.Shell(`kubectl -n mesh1-system delete secret cacerts`)
	util.Shell(`pushd ../testdata/examples/federation \
			&& export MESH1_KUBECONFIG=~/.kube/config \
			&& export MESH2_KUBECONFIG=~/.kube/config \
			&& ./cleanup.sh`)
	time.Sleep(time.Duration(20) * time.Second)
}

func TestSingleClusterFedDiffCert(t *testing.T) {
	defer cleanupSingleClusterFedDiffCert()

	t.Run("federation_single_cluster_different_cert_install", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Test federation install in a single cluster")
		util.Log.Info("Reference: https://github.com/maistra/istio/blob/maistra-2.1/pkg/servicemesh/federation/example/config-poc/install.sh")
		util.Log.Info("Running install_diff_cert.sh waiting 1 min...")
		util.Shell(`pushd ../testdata/examples/federation \
			&& export MESH1_KUBECONFIG=~/.kube/config \
			&& export MESH2_KUBECONFIG=~/.kube/config \
			&& ./install_diff_cert.sh`)

		util.Log.Info("Waiting 2 minutes...")
		time.Sleep(time.Duration(120) * time.Second)

		util.Log.Info("Verify mesh1 connection status")
		msg, err := util.Shell(`oc -n mesh1-system get servicemeshpeer mesh2 -o json`)
		if err != nil {
			t.Error("Failed to get servicemeshpeer in mesh1-system")
			util.Log.Error("Failed to get servicemeshpeer in mesh1-system")
		}
		if strings.Contains(msg, "\"connected\": true") {
			util.Log.Info("mesh1-system connected true")
		} else {
			t.Error("Failed to get mesh1-system connected")
			util.Log.Error("Failed to get mesh1-system connected")
		}

		util.Log.Info("Verify mesh2 connection status")
		msg, err = util.Shell(`oc -n mesh2-system get servicemeshpeer mesh1 -o json`)
		if err != nil {
			t.Error("Failed to get servicemeshpeer in mesh2-system")
			util.Log.Error("Failed to get servicemeshpeer in mesh2-system")
		}
		if strings.Contains(msg, "\"connected\": true") {
			util.Log.Info("mesh2-system connected true")
		} else {
			t.Error("Failed to get mesh2-system connected")
			util.Log.Error("Failed to get mesh2-system connected")
		}

		util.Log.Info("Verify if services from mesh1 are imported into mesh2")
		msg, err = util.Shell(`oc -n mesh2-system get importedservicesets mesh1 -o json`)
		if err != nil {
			t.Error("Failed to find services from mesh1 to mesh2")
			util.Log.Error("Failed to find services from mesh1 to mesh2")
		}
		if strings.Contains(msg, "mongodb.bookinfo.svc.mesh2-exports.local") && strings.Contains(msg, "ratings.bookinfo.svc.mesh2-exports.local") {
			util.Log.Info("mesh2-system gets both mongodb and ratings services from mesh1")
		} else {
			t.Error("mesh2-system failed to get both mongodb and ratings services from mesh1")
			util.Log.Error("mesh2-system failed to get both mongodb and ratings services from mesh1")
		}
	})
}

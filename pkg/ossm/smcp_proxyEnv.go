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
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupProxyEnv() {
	util.Log.Info("Cleanup ...")
	util.Shell(`kubectl -n istio-system patch smcp/basic --type=json -p='[{"op": "remove", "path": "/spec/proxy"}]'`)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
	util.KubeDeleteContents("bookinfo", testSpecProxyEnv)
	util.KubeDeleteContents("bookinfo", testAnnotationProxyEnv)
	time.Sleep(time.Duration(20) * time.Second)
}

func TestProxyEnv(t *testing.T) {
	defer cleanupProxyEnv()

	t.Run("smcp_test_annotation_proxyEnv", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Test annotation sidecar.maistra.io/proxyEnv")
		util.KubeApplyContents("bookinfo", testAnnotationProxyEnv)
		util.CheckPodRunning("bookinfo", "app=env")
		msg, err := util.ShellMuteOutput(`kubectl get po -n bookinfo -o yaml | grep maistra_test_env`)
		util.Inspect(err, "Failed to get variables", "", t)

		if strings.Contains(msg, "env_value") {
			util.Log.Info(msg)
		} else {
			t.Errorf("Failed to get env variable: %v", msg)
		}
	})

	t.Run("smcp_test_spec_proxyEnv", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Test SMCP .spec.proxy.runtime.container.env")
		if _, err := util.Shell(`kubectl -n istio-system patch smcp/basic --type=merge --patch="%s"`, ProxyEnvSMCPPath); err != nil {
			t.Fatal(err)
		}
		if _, err := util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Duration(40) * time.Second)

		util.KubeApplyContents("bookinfo", testSpecProxyEnv)
		util.CheckPodRunning("bookinfo", "app=env")
		msg, err := util.ShellMuteOutput(`kubectl get po -n bookinfo -o yaml | grep maistra_test_foo`)
		util.Inspect(err, "Failed to get variables", "", t)

		if strings.Contains(msg, "maistra_test_bar") {
			util.Log.Info(msg)
		} else {
			t.Errorf("Failed to get env variable: %v", msg)
		}
	})
}

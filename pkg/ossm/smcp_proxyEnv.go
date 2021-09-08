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
	util.KubeDeleteContents("bookinfo", testAnnotationProxyEnv)
	time.Sleep(time.Duration(20) * time.Second)
}

func TestProxyEnv(t *testing.T) {
	defer cleanupProxyEnv()

	t.Run("smcp_test_annotation_proxyEnv", func(t *testing.T) {
		defer util.RecoverPanic(t)

		util.Log.Info("Test annotation sidecar.maistra.io/proxyEnv")
		if getenv("SAMPLEARCH", "x86") == "p" {
			util.KubeApplyContents("bookinfo", testAnnotationProxyEnvP)
		} else if getenv("SAMPLEARCH", "x86") == "z" {
			util.KubeApplyContents("bookinfo", testAnnotationProxyEnvZ)
		} else {
			util.KubeApplyContents("bookinfo", testAnnotationProxyEnv)
		}
		util.CheckPodRunning("bookinfo", "app=env")
		msg, err := util.ShellMuteOutput(`kubectl get po -n bookinfo -o yaml | grep maistra_test_env`)
		util.Inspect(err, "Failed to get variables", "", t)

		if strings.Contains(msg, "env_value") {
			util.Log.Info(msg)
		} else {
			t.Errorf("Failed to get env variable: %v", msg)
		}
	})
}

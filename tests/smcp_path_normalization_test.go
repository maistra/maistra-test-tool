// Copyright 2020 Red Hat, Inc.
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

package tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupPathNormalizationSMCP() {
	log.Info("# Cleanup ...")
	util.KubeDelete("foo", httpbinPathResource, kubeconfig)
	cleanHttpbin("foo")
	cleanSleep("foo")
	util.Shell(`kubectl patch -n %s smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/techPreview"}]'`, meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*8) * time.Second)
	// avoid namespace recreation for downstream service account settings
}

func TestPathNormalizationSMCP(t *testing.T) {
	defer cleanupPathNormalizationSMCP()

	t.Run("Operator_test_smcp_path_normalization", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Update SMCP pathNormalization")
		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{"spec":{"techPreview":{"global":{"pathNormalization":"DECODE_AND_MERGE_SLASHES"}}}}'`, meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*4) * time.Second)

		deployHttpbin("foo")
		deploySleep("foo")

		util.CheckPodRunning("foo", "app=httpbin", kubeconfig)
		util.KubeApply("foo", httpbinPathResource, kubeconfig)

		log.Info("Verify setup")

		sleepPod, err := util.GetPodName("foo", "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -s -o /dev/null -w "sleep.%s to httpbin.%s: %%{http_code}"`,
			"foo", "foo", "foo")
		msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if !strings.Contains(msg, "200") {
			log.Errorf("Verify setup -- Unexpected response code: %s", msg)
		} else {
			log.Infof("Success. Get expected response: %s", msg)
		}
	})
}

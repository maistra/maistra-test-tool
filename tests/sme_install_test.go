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
	"maistra/util"
	"strings"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanUpTestExtensionInstall(namespace string) {
	log.Info("# Cleanup ...")
	cleanHttpbin(testNamespace)
	cleanSleep(testNamespace)
	util.KubeDeleteContents(testNamespace, httpbinServiceMeshExtension, kubeconfig)

	util.Shell(`kubectl patch -n %s smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/techPreview"}]'`, meshNamespace, smcpName)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func TestExtensionInstall(t *testing.T) {
	defer cleanUpTestExtensionInstall(testNamespace)

	t.Run("Operator_test_sme_install", func(t *testing.T) {

		defer recoverPanic(t)

		log.Info("Enable Extension support")
		util.Shell(`kubectl patch -n %s smcp/%s --type merge -p '{%s:{%s}}}}'`,
			meshNamespace, smcpName,
			`"spec":{"techPreview":{"wasmExtensions"`,
			`"enabled": true`)
		util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)

		time.Sleep(time.Duration(waitTime) * time.Second)
		util.CheckPodRunning(meshNamespace, "app=wasm-cacher", kubeconfig)

		log.Info("Creating ServiceMeshExtension")
		util.KubeApplyContents(testNamespace, httpbinServiceMeshExtension, kubeconfig)

		log.Info("Deploy httpbin pod")
		deployHttpbin(testNamespace)

		log.Info("Deploy sleep pod")
		deploySleep(testNamespace)

		time.Sleep(time.Duration(waitTime*2) * time.Second)
		pod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "failed to get sleep pod", "", t)

		command := "curl -i httpbin:8000/headers"
		msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfig)
		if !strings.Contains(msg, "custom-header: test") {
			t.Errorf("custom-header not present: Expected value 'test'")
			log.Errorf("custom-header not present: Expected value 'test'")
		}
	})
}

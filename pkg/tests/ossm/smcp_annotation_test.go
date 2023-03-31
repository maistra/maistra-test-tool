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
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/deployment-testenv-x86.yaml
	testenvDeploymentX86 string

	//go:embed yaml/deployment-testenv-z.yaml
	testenvDeploymentZ string

	//go:embed yaml/deployment-testenv-p.yaml
	testenvDeploymentP string

	InjectedAnnotationsSMCPPatch = `
spec:
  proxy:
    injection:
      autoInject: true
      injectedAnnotations:
        test1.annotation-from-smcp: test1
        test2.annotation-from-smcp: '["test2"]'
        test3.annotation-from-smcp: '{test3}'
`
)

func cleanupSMCPAnnotations() {
	log.Log.Info("Cleanup ...")
	util.KubeDeleteContents("bookinfo", testenvDeploymentX86)
	time.Sleep(time.Duration(20) * time.Second)
}

func TestSMCPAnnotations(t *testing.T) {
	test.NewTest(t).Id("T29").Groups(test.Full).NotRefactoredYet()

	defer cleanupSMCPAnnotations()

	t.Run("smcp_test_annotation_proxyEnv", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Test annotation sidecar.maistra.io/proxyEnv")
		if env.Getenv("SAMPLEARCH", "x86") == "p" {
			util.KubeApplyContents("bookinfo", testenvDeploymentP)
		} else if env.Getenv("SAMPLEARCH", "x86") == "z" {
			util.KubeApplyContents("bookinfo", testenvDeploymentZ)
		} else {
			util.KubeApplyContents("bookinfo", testenvDeploymentX86)
		}
		util.CheckPodRunning("bookinfo", "app=env")
		msg, err := util.ShellMuteOutput(`kubectl get po -n bookinfo -o yaml | grep maistra_test_env`)
		util.Inspect(err, "Failed to get variables", "", t)

		if strings.Contains(msg, "env_value") {
			log.Log.Info(msg)
		} else {
			t.Errorf("Failed to get env variable: %v", msg)
		}
		util.KubeDeleteContents("bookinfo", testenvDeploymentX86)
		time.Sleep(time.Duration(30) * time.Second)
	})

	t.Run("smcp_test_annotation_quote_injection", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Test SMCP annotation quote value injection")
		if _, err := util.Shell(`kubectl -n %s patch smcp/%s --type=merge --patch="%s"`, meshNamespace, smcpName, InjectedAnnotationsSMCPPatch); err != nil {
			t.Fatal(err)
		}

		if _, err := util.Shell(`oc -n %s wait --for condition=Ready smcp/%s --timeout 180s`, meshNamespace, smcpName); err != nil {
			t.Fatal(err)
		}

		log.Log.Info("Check a pod annotations")
		if env.Getenv("SAMPLEARCH", "x86") == "p" {
			util.KubeApplyContents("bookinfo", testenvDeploymentP)
		} else if env.Getenv("SAMPLEARCH", "x86") == "z" {
			util.KubeApplyContents("bookinfo", testenvDeploymentZ)
		} else {
			util.KubeApplyContents("bookinfo", testenvDeploymentX86)
		}
		util.CheckPodRunning("bookinfo", "app=env")
		podName, err := util.GetPodName("bookinfo", "app=env")
		if err != nil {
			t.Fatalf("Failed to get pod name: %v", err)
		}
		annotations, err := util.GetPodAnnotations("bookinfo", podName, 30)
		if err != nil {
			t.Fatalf("Failed to get annotations: %v", err)
		}
		log.Log.Infof("Checking annotation: %v", annotations["test1.annotation-from-smcp"])
		if _, ok := annotations["test1.annotation-from-smcp"]; !ok {
			t.Errorf("Failed to get annotations: test1.annotation-from-smcp: test1")
		} else if annotations["test1.annotation-from-smcp"] != "test1" {
			t.Errorf("Failed to get annotations: test1.annotation-from-smcp: test1")
		}
		log.Log.Infof("Checking annotation: %v", annotations["test2.annotation-from-smcp"])
		if _, ok := annotations["test2.annotation-from-smcp"]; !ok {
			t.Errorf("Failed to get annotations: test2.annotation-from-smcp: '[test2]'")
		} else if annotations["test2.annotation-from-smcp"] != "[test2]" {
			t.Errorf("Failed to get annotations: test2.annotation-from-smcp: '[test2]'")
		}
		log.Log.Infof("Checking annotation: %v", annotations["test3.annotation-from-smcp"])
		if _, ok := annotations["test3.annotation-from-smcp"]; !ok {
			t.Errorf("Failed to get annotations: test3.annotation-from-smcp: '{test3}'")
		} else if annotations["test3.annotation-from-smcp"] != "{test3}" {
			t.Errorf("Failed to get annotations: test3.annotation-from-smcp: '{test3}'")
		}
	})
}

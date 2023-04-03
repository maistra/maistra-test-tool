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

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

const (
	initContainerNS         = "bookinfo"
	initContainerGoldString = "init worked"
)

var (
	//go:embed yaml/deployment-sleep-init.yaml
	testInitContainerYAML string
)

func cleanupInitContainer() {
	util.KubeDeleteContents(initContainerNS, testInitContainerYAML)
}

func TestInitContainer(t *testing.T) {
	test.NewTest(t).Id("T33").Groups(test.Full).NotRefactoredYet()

	defer cleanupInitContainer()

	if err := util.KubeApplyContents(initContainerNS, testInitContainerYAML); err != nil {
		t.Fatalf("error creating the pod: %v", err)
	}
	if err := util.CheckPodRunning(initContainerNS, "app=sleep-init"); err != nil {
		t.Fatalf("sleep-init pod is not running: %v", err)
	}

	logs := util.GetPodLogsForLabel(initContainerNS, "app=sleep-init", "init", false, false)
	if !strings.Contains(logs, initContainerGoldString) {
		t.Fatalf("expected init container log to contain the string %q, but got %q", initContainerGoldString, logs)
	}
}

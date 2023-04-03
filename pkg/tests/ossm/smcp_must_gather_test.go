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
	"path/filepath"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var mustGatherImage = "registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel8"
var mustGatherTag string = env.Getenv("MUSTGATHERTAG", "2.3")

func cleanupMustGatherTest() {
	log.Log.Info("Cleanup ...")
	bookinfo := examples.Bookinfo{"bookinfo"}
	bookinfo.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestMustGather(t *testing.T) {
	test.NewTest(t).Id("T30").Groups(test.Full).NotRefactoredYet()

	defer cleanupMustGatherTest()
	defer util.RecoverPanic(t)

	log.Log.Info("Deploy bookinfo in bookinfo ns")
	bookinfo := examples.Bookinfo{"bookinfo"}
	bookinfo.Install(false)

	t.Run("smcp_test_must_gather", func(t *testing.T) {
		defer util.RecoverPanic(t)
		log.Log.Info("Test must-gather log collection")
		log.Log.Info("Must-gather image: ", mustGatherImage, ":", mustGatherTag)
		util.Shell(`mkdir -p debug; oc adm must-gather --dest-dir=./debug --image=%s:%s`, mustGatherImage, mustGatherTag)

		log.Log.Info("Check cluster-scoped openshift-operators.servicemesh-resources.maistra.io.yaml")
		pattern := "debug/*must-gather*/cluster-scoped-resources/admissionregistration.k8s.io/mutatingwebhookconfigurations/openshift-operators.servicemesh-resources.maistra.io.yaml"
		matches, err := filepath.Glob(pattern)
		if err != nil || len(matches) == 0 {
			log.Log.Errorf("openshift-operators.servicemesh-resources.maistra.io.yaml file not found: %s", matches)
			t.Errorf("openshift-operators.servicemesh-resources.maistra.io.yaml file not found: %s", matches)
		} else {
			log.Log.Infof("file exists: %s", matches)
		}
	})
}

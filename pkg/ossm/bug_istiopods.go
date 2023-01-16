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
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupIstioPodsTest() {
	util.Log.Info("Cleanup ...")
	bookinfo := examples.Bookinfo{"bookinfo"}
	bookinfo.Uninstall()
	util.Shell(`../scripts/smmr/clean_members_50.sh`)
	time.Sleep(time.Duration(20) * time.Second)
}

// TestIstioPodProbesFails tests that Istio pod get stuck with probes failure after restart. Jira ticket: https://issues.redhat.com/browse/OSSM-2434
func TestIstioPodProbesFails(t *testing.T) {
	defer cleanupIstioPodsTest()
	defer util.RecoverPanic(t)

	util.Log.Info("Deploy bookinfo in bookinfo ns")
	bookinfo := examples.Bookinfo{"bookinfo"}
	bookinfo.Install(false)

	t.Run("smcp_test_istio_pod_probes_failure", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.Log.Info("Testing: Istio Pod get stuck with probes failure after restart")
		util.Log.Info("Create 50 new namespaces")
		util.Shell(`../scripts/smmr/create_members_50.sh`)
		util.Log.Info("Namespaces created...")
		rand.Seed(time.Now().UnixNano())
		// Random number of deletes for the pod between 4 and 10
		randId := rand.Intn(10-4) + 4
		util.Log.Info("Random number of deletes: ", randId)
		count := 0
		for count < randId {
			util.Log.Info("*** Get the istiod pod name")
			istiod, _ := util.GetPodName(`istio-system`, `app=istiod`)
			util.Log.Info("*** Delete Istio pod: ", istiod)
			util.Shell(`oc delete pod %s -n istio-system`, istiod)
			deleted, _ := util.CheckPodDeletion(`istio-system`, `app=istiod`, istiod, 60)
			if deleted {
				util.Log.Info("*** Istiod pod deleted: ", istiod)
				ready := util.CheckPodRunning(`istio-system`, `app=istiod`)
				if ready == nil {
					util.Log.Info("*** New Istiod pod is running")
					status, _ := util.Shell(`oc get pods -n istio-system | grep istiod | awk '{print $2}'`)
					istiod, _ = util.GetPodName(`istio-system`, `app=istiod`)
					if strings.Contains(status, `1/1`) {
						util.Log.Info("*** Istiod pod is running: ", istiod)
					} else {
						//Get events that are not type = Normal from the pod
						event, _ := util.Shell(`kubectl get events  -n istio-system --field-selector type!=Normal,involvedObject.name=%s |tail -1`, istiod)
						if strings.Contains(event, "Readiness probe failed") {
							t.Fatalf("****** Istio pod is not running and fail because of readiness probe failure: %s", event)
						} else if event == "No resources found in istio-system namespace." {
							util.Log.Info("*** No eventss for pod: ", istiod)
						} else {
							t.Errorf("*** Istio pod is not running but is not failing because of readiness probe failure: %s", event)
						}
					}
					count++
				}
			} else {
				t.Errorf("*** Istiod pod is not deleted: %s", istiod)
			}
		}
		util.Log.Info("*** Test finished without errors after deletion of pods")
	})
}
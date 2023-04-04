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

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

// TestIstioPodProbesFails tests that Istio pod get stuck with probes failure after restart. Jira ticket bug: https://issues.redhat.com/browse/OSSM-2434
func TestIstioPodProbesFails(t *testing.T) {
	NewTest(t).Id("T35").Groups(Full).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		ns := "bookinfo"
		data := map[string]string{
			"Count":     "50",
			"Namespace": meshNamespace,
		}

		t.Cleanup(func() {
			//need to define wich cleanup is needed
			// log.Log.Info("Cleanup ...")
			// bookinfo := examples.Bookinfo{"bookinfo"}
			// bookinfo.Uninstall()
			// util.Shell(fmt.Sprintf(env.GetRootDir()+"/scripts/smmr/clean_members.sh %d", numberOfNamespaces))
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStep("Install Bookinfo application")
		app.InstallAndWaitReady(t, app.Bookinfo(ns))

		t.LogStep("Create Namespaces and SMMR")
		oc.CreateNamespaces(t, multiple_namespaces, data)
		oc.UpdateSMMRMultipleNamespaces(t, multiple_smmr, data)

	})

	t.Run("smcp_test_istio_pod_probes_failure", func(t *testing.T) {
		log.Log.Info("Namespaces created...")
		rand.Seed(time.Now().UnixNano())
		// Random number of deletes for the pod between 4 and 10
		randId := rand.Intn(10-4) + 4
		log.Log.Info("Random number of deletes: ", randId)
		count := 0
		for count < randId {
			log.Log.Info("*** Get the istiod pod name")
			istiod, _ := util.GetPodName(`istio-system`, `app=istiod`)
			log.Log.Info("*** Delete Istio pod: ", istiod)
			util.Shell(`oc delete pod %s -n istio-system`, istiod)
			deleted, _ := util.CheckPodDeletion(`istio-system`, `app=istiod`, istiod, 60)
			if deleted {
				log.Log.Info("*** Istiod pod deleted: ", istiod)
				running := util.CheckPodRunning(`istio-system`, `app=istiod`)
				if running == nil {
					log.Log.Info("*** New Istiod pod is running")
					ready, _ := util.CheckPodReady(`istio-system`, `app=istiod`, 10)
					if ready {
						log.Log.Info("*** New Istiod pod is ready")
					} else {
						istiod, _ = util.GetPodName(`istio-system`, `app=istiod`)
						// Get events that are not type = Normal from the pod
						event, _ := util.Shell(`kubectl get events  -n istio-system --field-selector type!=Normal,involvedObject.name=%s |tail -1`, istiod)
						if strings.Contains(event, "Readiness probe failed") {
							t.Fatalf("****** Istio pod is not running and fail because of readiness probe failure: %s", event)
						} else if event == "No resources found in istio-system namespace." {
							log.Log.Info("*** No eventss for pod: ", istiod)
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
		log.Log.Info("*** Test finished without errors after deletion of pods")
	})
}

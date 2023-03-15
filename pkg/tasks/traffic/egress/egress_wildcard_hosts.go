// Copyright 2021 Red Hat, Inc.
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

package egress

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

func cleanupEgressWildcard() {
	log.Log.Info("Cleanup")
	util.KubeDeleteContents("bookinfo", util.RunTemplate(EgressWildcardGatewayTemplate, smcp))
	util.KubeDeleteContents("bookinfo", EgressWildcardEntry)
	sleep := examples.Sleep{"bookinfo"}
	sleep.Uninstall()
	time.Sleep(time.Duration(20) * time.Second)
}

func TestEgressWildcard(t *testing.T) {
	defer cleanupEgressWildcard()
	defer util.RecoverPanic(t)

	log.Log.Info("Test Egress Wildcard Hosts")
	sleep := examples.Sleep{"bookinfo"}
	sleep.Install()
	sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_direct_traffic_wildcard_host", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Configure direct traffic to a wildcard host")
		util.KubeApplyContents("bookinfo", EgressWildcardEntry)
		time.Sleep(time.Duration(10) * time.Second)

		command := `curl -s https://en.wikipedia.org/wiki/Main_Page | grep -o "<title>.*</title>"; curl -s https://de.wikipedia.org/wiki/Wikipedia:Hauptseite | grep -o "<title>.*</title>"`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "<title>Wikipedia, the free encyclopedia</title>\n<title>Wikipedia – Die freie Enzyklopädie</title>") {
			log.Log.Infof("Success. Got Wikipedia response: %s", msg)
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.KubeDeleteContents("bookinfo", EgressWildcardEntry)
	})

	t.Run("TrafficManagement_egress_gateway_wildcard_host", func(t *testing.T) {
		defer util.RecoverPanic(t)

		log.Log.Info("Configure egress gateway to a wildcard host")
		util.KubeApplyContents("bookinfo", util.RunTemplate(EgressWildcardGatewayTemplate, smcp))
		time.Sleep(time.Duration(10) * time.Second)

		command := `curl -s https://en.wikipedia.org/wiki/Main_Page | grep -o "<title>.*</title>"; curl -s https://de.wikipedia.org/wiki/Wikipedia:Hauptseite | grep -o "<title>.*</title>"`
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "<title>Wikipedia, the free encyclopedia</title>\n<title>Wikipedia – Die freie Enzyklopädie</title>") {
			log.Log.Infof("Success. Got Wikipedia response: %s", msg)
		} else {
			log.Log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.KubeDeleteContents("bookinfo", util.RunTemplate(EgressWildcardGatewayTemplate, smcp))
	})

	// setup SNI proxy for wildcard arbitrary domains
}

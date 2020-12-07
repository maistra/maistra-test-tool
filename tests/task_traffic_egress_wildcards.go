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
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupEgressGateways(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, egressWildcardGateway, kubeconfig)
	cleanSleep(namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func TestEgressWildcards(t *testing.T) {
	defer cleanupEgressGateways(testNamespace)
	defer recoverPanic(t)

	log.Info("# TestEgressWildcards")

	deploySleep(testNamespace)
	sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_wildcard", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("create a Gateway to external wikipedia.org")
		util.KubeApplyContents(testNamespace, egressWildcardGateway, kubeconfig)
		// OCP Route created by ior
		time.Sleep(time.Duration(waitTime) * time.Second)
		command := "curl -sL -o /dev/null -D - curl -s https://en.wikipedia.org/wiki/Main_Page | grep -o \"<title>.*</title>\"; curl -s https://de.wikipedia.org/wiki/Wikipedia:Hauptseite | grep -o \"<title>.*</title>\""

		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "<title>Wikipedia, the free encyclopedia</title><title>Wikipedia – Die freie Enzyklopädie</title>") {
			log.Infof("Success. Got Wikipedia response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.KubeDeleteContents(testNamespace, egressWildcardGateway, kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)
	})

}

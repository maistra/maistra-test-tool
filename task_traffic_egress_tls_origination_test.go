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

package main

import (
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupEgressTLSOrigination(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, cnnextServiceEntryTLS, kubeconfig)
	util.KubeDeleteContents(namespace, cnnextServiceEntry, kubeconfig)
	cleanSleep(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestEgressTLSOrigination(t *testing.T) {
	defer cleanupEgressTLSOrigination(testNamespace)
	defer recoverPanic(t)

	log.Info("# TestEgressTLSOrigination")
	deploySleep(testNamespace)
	sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_configure_access_to_external_service", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("create a ServiceEntry to external edition.cnn.com")
		util.KubeApplyContents(testNamespace, cnnextServiceEntry, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		command := "curl -sL -o /dev/null -D - http://edition.cnn.com/politics"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "301") {
			log.Infof("Success. Get http://edition.cnn.com/politics response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})

	t.Run("TrafficManagement_egress_tls_origination", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("refine a ServiceEntry to external edition.cnn.com")
		util.KubeApplyContents(testNamespace, cnnextServiceEntryTLS, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		command := "curl -sL -o /dev/null -D - http://edition.cnn.com/politics"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200") {
			log.Infof("Success. Get http://edition.cnn.com/politics response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		command = "curl -sL -o /dev/null -D - https://edition.cnn.com/politics"
		msg, err = util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200") {
			log.Infof("Success. Get https://edition.cnn.com/politics response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})
}

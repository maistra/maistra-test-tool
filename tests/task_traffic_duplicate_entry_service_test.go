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

func cleanupDuplicateEntryService() {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(testNamespace, DuplicateEntryService, kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func TestDuplicateEntryService(t *testing.T) {
	defer cleanupDuplicateEntryService()
	defer recoverPanic(t)

	log.Info("# Test VirtualService and Service Duplicate entry of domain")
	util.KubeApplyContents(testNamespace, DuplicateEntryService, kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)

	log.Info("Check istiod logs")
	pod, err := util.GetPodName(meshNamespace, "app=istiod", kubeconfig)
	util.Inspect(err, "failed to get istiod pod", "", t)
	msg, _ := util.ShellMuteOutput(`kubectl -n istio-system logs %s`, pod)
	if strings.Contains(msg, "Duplicate entry of domain") {
		t.Errorf("istiod log Duplicate entry of domain ERROR after applying VirtualService")
		util.Shell(`kubectl -n istio-system logs %s | grep "Duplicate entry of domain"`, pod)
	}
}

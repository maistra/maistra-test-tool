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
	"path/filepath"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupMustGatherTest(namespace string) {
	log.Info("# Cleanup ...")
	util.ShellMuteOutput("rm -rf ./debug")
	cleanBookinfo(namespace)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestMustGather(t *testing.T) {
	defer cleanupMustGatherTest(testNamespace)
	defer recoverPanic(t)

	log.Info("Deploy bookinfo in bookinfo ns")
	deployBookinfo(testNamespace, false)

	log.Info("Test must-gather log collection")
	msg, err := util.Shell("mkdir -p debug; oc adm must-gather --dest-dir=./debug --image=%s", mustGatherImage)
	log.Info("Check CLI output message")
	if err != nil || strings.Contains(msg, "error") {
		log.Errorf("must-gather command error: %s", msg)
		t.Errorf("must-gather command error: %s", msg)
	}

	log.Info("Check cluster-scoped openshift-operators.servicemesh-resources.maistra.io.yaml")
	pattern := "debug/*must-gather*/cluster-scoped-resources/admissionregistration.k8s.io/mutatingwebhookconfigurations/openshift-operators.servicemesh-resources.maistra.io.yaml"
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		log.Errorf("openshift-operators.servicemesh-resources.maistra.io.yaml file not found: %s", matches)
		t.Errorf("openshift-operators.servicemesh-resources.maistra.io.yaml file not found: %s", matches)
	} else {
		log.Infof("file exists: %s", matches)
	}

	log.Info("Check namespaces bookinfo bookinfo.yaml")
	pattern = "debug/*must-gather*/namespaces/bookinfo/bookinfo.yaml"
	matches, err = filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		log.Errorf("bookinfo.yaml file not found: %s", matches)
		t.Errorf("bookinfo.yaml file not found: %s", matches)
	} else {
		log.Infof("file exists: %s", matches)
	}
}

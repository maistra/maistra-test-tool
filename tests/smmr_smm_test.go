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
	"maistra/util"
	"strings"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanupTestServiceMember() {
	log.Info("# Cleanup ...")
	util.KubeDelete(testNamespace, smmDefault, kubeconfig)
	util.KubeDelete(meshNamespace, smmrTest, kubeconfig)
	util.KubeApply(meshNamespace, smmrTest, kubeconfig)
	time.Sleep(time.Duration(waitTime*2) * time.Second)
}

func TestServiceMember(t *testing.T) {
	defer cleanupTestServiceMember()

	t.Run("SMMR_Create", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("Create SMMR")
		if err := util.KubeApply(meshNamespace, smmrTest, kubeconfig); err != nil {
			t.Errorf("Failed to create SMMR")
			log.Errorf("Failed to create SMMR")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("SMMR_Delete", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("Delete SMMR")
		if err := util.KubeDelete(meshNamespace, smmrTest, kubeconfig); err != nil {
			t.Errorf("Failed to delete SMMR")
			log.Errorf("Failed to delete SMMR")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("SMM_Create", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("Create a bookinfo SMM")
		if err := util.KubeApply(testNamespace, smmDefault, kubeconfig); err != nil {
			t.Errorf("Failed to create SMM in bookinfo ns")
			log.Errorf("Failed to create SMM in bookinfo ns")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("SMM_Delete", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("Delete a bookinfo SMM")
		if err := util.KubeDelete(testNamespace, smmDefault, kubeconfig); err != nil {
			t.Errorf("Failed to delete SMM in bookinfo ns")
			log.Errorf("Failed to delete SMM in bookinfo ns")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("SMMR_SMM_Create", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("Create a SMM which has been included in SMMR")
		util.KubeApply(meshNamespace, smmrTest, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		if err := util.KubeApply(testNamespace, smmDefault, kubeconfig); err != nil {
			t.Errorf("Failed to create SMM in bookinfo ns")
			log.Errorf("Failed to create SMM in bookinfo ns")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)
	})

	t.Run("SMMR_SMM_Delete", func(t *testing.T) {
		defer recoverPanic(t)
		log.Info("Delete a SMM which has been included in SMMR")
		util.KubeApply(meshNamespace, smmrTest, kubeconfig)
		time.Sleep(time.Duration(waitTime) * time.Second)

		if err := util.KubeDelete(testNamespace, smmDefault, kubeconfig); err != nil {
			t.Errorf("Failed to delete SMM in bookinfo ns")
			log.Errorf("Failed to delete SMM in bookinfo ns")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		log.Info("Check SMMR: mesh users should not be able to remove the project from the mesh by removing the SMM object")
		msg, _ := util.ShellMuteOutput(`kubectl get -n %s smmr/default -o yaml`, meshNamespace)
		if strings.Contains(msg, "- "+testNamespace) {
			log.Info(string(msg))
		} else {
			t.Errorf("Failed to find project in SMMR: %v", testNamespace)
			log.Info(string(msg))
		}
	})
	time.Sleep(time.Duration(waitTime) * time.Second)
}

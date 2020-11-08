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
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanupInvalidSMCPFields(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, invalidSMCPFields, kubeconfig)
	time.Sleep(time.Duration(waitTime*8) * time.Second)
	// avoid namespace recreation for downstream service account settings
}

func TestInvalidSMCPFields(t *testing.T) {
	defer cleanupInvalidSMCPFields("istio-invalid")

	t.Run("Operator_test_smcp_invalid_fields", func(t *testing.T) {

		defer recoverPanic(t)

		// create a smcp with nil redudant policies field for Jaeger - See MAISTRA-1983
		util.CreateOCPNamespace("istio-invalid", kubeconfig)
		log.Info("Update SMCP with 2.0.0 invalid fields. They should pass on 2.0.0-1 and above")
		if err := util.KubeApply("istio-invalid", invalidSMCPFields, kubeconfig); err != nil {
			t.Errorf("Failed to deploy SMCP with nil fields")
			log.Errorf("Failed to deploy SMCP with nil fields")
		}
		time.Sleep(time.Duration(waitTime*2) * time.Second)

	})
}

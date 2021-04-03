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

func cleanupExcludeOutboundPortsAnnotation(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDelete(namespace, excludeOutboundPortsAnnotation, kubeconfig)
	time.Sleep(time.Duration(waitTime*8) * time.Second)
	util.DeleteOCPNamespace(namespace,kubeconfig)
}

func TestExcludeOutboundPortsAnnotation(t *testing.T) {
	defer cleanupExcludeOutboundPortsAnnotation("exclude-outboundports-annotation")

	t.Run("Operator_test_excludeOutboundPortsAnnotation", func(t *testing.T) {

		defer recoverPanic(t)

		util.CreateOCPNamespace("exclude-outboundports-annotation", kubeconfig)
		log.Info("Automation for MAISTRA-2134. The annotation should pass on 2.0.3 and above")
		if err := util.KubeApply("exclude-outboundports-annotation", excludeOutboundPortsAnnotation, kubeconfig); err != nil {
			t.Errorf("Failed to deploy HTTP bin with traffic.sidecar.istio.io/excludeOutboundPorts annotation")
		}
		util.CheckPodRunning("exclude-outboundports-annotation", "app=httpbin", kubeconfig)


		time.Sleep(time.Duration(waitTime*2) * time.Second)

	})
}

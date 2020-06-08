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

func cleanupCitadelSecretGeneration(namespace string) {
	log.Info("# Cleanup ...")

	util.Shell("kubectl delete project foo")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"security\":{\"enableNamespacesByDefault\":true}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=citadel", kubeconfig)
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":false,\"mtls\":{\"enabled\":false}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)
	util.Shell("oc new-project foo")
}

func TestCitadelSecretGeneration(t *testing.T) {
	defer cleanupCitadelSecretGeneration("foo")
	defer recoverPanic(t)

	log.Info("Configure Citadel Service Account Secret Generation")
	// update mtls to true
	log.Info("Update SMCP mtls to true")
	util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"global\":{\"controlPlaneSecurityEnabled\":true,\"mtls\":{\"enabled\":true}}}}}'", meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime*10) * time.Second)
	util.CheckPodRunning(meshNamespace, "istio=galley", kubeconfig)

	log.Info("Check existing istio secrets in foo")
	util.Shell("kubectl get secrets -n foo | grep istio.io")

	t.Run("Security_citadel_deactivating_secret_generation", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("label the namespace foo")
		_, err := util.Shell("kubectl label ns foo ca.istio.io/override=false")
		if err != nil {
			log.Infof("OCP4 permission forbidden. Need cluster admin user to run label ns command.")
		} else {
			util.KubeApplyContents("foo", fooSampleSA, kubeconfig)
			time.Sleep(time.Duration(waitTime*2) * time.Second)
			msg, err := util.Shell("kubectl get secrets -n foo | grep istio.io")
			util.Inspect(err, "Failed to get sa", "", t)
			if strings.Contains(msg, "istio.sample-service-account") {
				t.Errorf("Deactivating failed. Result: \n%s", msg)
				log.Errorf("Deactivating failed. Result: \n%s", msg)
			} else {
				log.Infof("Success. Get expected result: \n%s", msg)
			}
		}
	})

	t.Run("Security_citadel_opt-in_secret_generation", func(t *testing.T) {
		defer recoverPanic(t)

		util.Shell("kubectl delete project foo")
		log.Info("Redeploy Citadel")
		util.ShellMuteOutput("kubectl patch -n %s smcp/%s --type merge -p '{\"spec\":{\"istio\":{\"security\":{\"enableNamespacesByDefault\":false}}}}'", meshNamespace, smcpName)
		time.Sleep(time.Duration(waitTime*10) * time.Second)
		util.CheckPodRunning(meshNamespace, "istio=citadel", kubeconfig)

		util.Shell("oc new-project foo")
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		msg, err := util.Shell("kubectl get secrets -n foo | grep istio.io")
		if err == nil || strings.Contains(msg, "istio.default") {
			t.Errorf("Opt-in failed. Result: \n%s", msg)
			log.Errorf("Opt-in failed. Result: \n%s", msg)
		} else {
			log.Infof("Success. Get expected result: \n%s", msg)
		}

		log.Info("Override generation for the foo namespace")
		_, err = util.Shell("kubectl label ns foo ca.istio.io/override=true")
		if err != nil {
			log.Infof("OCP4 permission forbidden. Need cluster admin user to run label ns command.")
		} else {
			util.KubeApplyContents("foo", fooSampleSA, kubeconfig)
			time.Sleep(time.Duration(waitTime*2) * time.Second)
			msg, err = util.Shell("kubectl get secrets -n foo | grep istio.io")
			util.Inspect(err, "Failed to get sa", "", t)
			if !strings.Contains(msg, "istio.sample-service-account") {
				t.Errorf("Override failed. Result: \n%s", msg)
				log.Errorf("Override failed. Result: \n%s", msg)
			} else {
				log.Infof("Success. Get expected result: \n%s", msg)
			}
		}
	})
}

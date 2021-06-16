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
	"context"
	"fmt"
	"maistra/util"
	"strings"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanUpTestExtensionInstall(namespace string) {
	log.Info("# Cleanup ...")
	cleanHttpbin(testNamespace)
	cleanSleep(testNamespace)
	util.KubeDeleteContents(testNamespace, httpbinServiceMeshExtension, kubeconfig)

	util.Shell(`kubectl patch -n %s smcp/%s --type=json -p='[{"op": "remove", "path": "/spec/techPreview"}]'`, meshNamespace, smcpName)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func TestExtensionInstall(t *testing.T) {
	defer cleanUpTestExtensionInstall(testNamespace)

	t.Run("Operator_test_sme_install", func(t *testing.T) {

		defer recoverPanic(t)

		util.CheckPodRunning(meshNamespace, "app=wasm-cacher", kubeconfig)

		log.Info("Creating ServiceMeshExtension")
		util.KubeApplyContents(testNamespace, httpbinServiceMeshExtension, kubeconfig)

		log.Info("Deploy httpbin pod")
		deployHttpbin(testNamespace)

		log.Info("Deploy sleep pod")
		deploySleep(testNamespace)

		log.Info("")
		pod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "failed to get sleep pod", "", t)

		if err := checkSMEReady(testNamespace, "header-append", kubeconfig); err != nil {
			t.Fatalf("error checking for SME header-append: %v", err)
		}

		command := "curl -I httpbin:8000/headers"
		msg, err := util.PodExec(testNamespace, pod, "sleep", command, false, kubeconfig)
		if err != nil {
			t.Fatalf("error running command %q in pod %q: %v", command, pod, err)
		}
		if !strings.Contains(msg, "maistra: rocks") {
			t.Fatalf("custom header not present: Expected value 'maistra: rocks'")
		}
	})
}

func checkSMEReady(n, name string, kubeconfig string) error {
	retry := util.Retrier{
		BaseDelay: 30 * time.Second,
		MaxDelay:  30 * time.Second,
		Retries:   6,
	}

	retryFn := func(_ context.Context, i int) error {
		ready, err := isSMEReady(n, "header-append", kubeconfig)
		if err != nil {
			return err
		}
		if ready != "true" {
			return fmt.Errorf("sme is not ready")
		}
		return nil
	}

	ctx := context.Background()
	_, err := retry.Retry(ctx, retryFn)
	return err
}

func isSMEReady(ns, name, kubeconfig string) (string, error) {
	res, err := util.Shell("kubectl -n %s get sme %s -o jsonpath='{.status.deployment.ready}' --kubeconfig=%s", ns, name, kubeconfig)
	if err != nil {
		return "", fmt.Errorf("failed to get SME status for %s/%s: %v", ns, name, err)
	}
	return strings.Trim(res, "'"), nil
}

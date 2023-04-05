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

package ossm

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

//go:embed yaml/servicemeshextension-header-append.yaml
var httpbinServiceMeshExtension string

func cleanUpTestExtensionInstall() {
	httpbinPod, err := util.GetPodName("bookinfo", "app=httpbin")
	if err == nil {
		log.Log.Info("# httpbin proxy log: ")
		log.Log.Info(util.GetPodLogs("bookinfo", httpbinPod, "istio-proxy", false, false))
		log.Log.Info("# end of httpbin proxy log")
	}

	log.Log.Info("# Cleanup ...")
	httpbin := examples.Httpbin{Namespace: "bookinfo"}
	sleep := examples.Sleep{Namespace: "bookinfo"}
	httpbin.Uninstall()
	sleep.Uninstall()
	util.KubeDeleteContents("bookinfo", httpbinServiceMeshExtension)
	time.Sleep(time.Duration(20) * time.Second)
}

func TestExtensionInstall(t *testing.T) {
	test.NewTest(t).Groups(test.Full).NotRefactoredYet()
	t.Skip()

	defer cleanUpTestExtensionInstall()
	httpbin := examples.Httpbin{Namespace: "bookinfo"}
	sleep := examples.Sleep{Namespace: "bookinfo"}
	log.Log.Info("Deploy httpbin pod")
	httpbin.Install()
	log.Log.Info("Deploy sleep pod")
	sleep.Install()
	sleepPod, err := util.GetPodName("bookinfo", "app=sleep")
	util.Inspect(err, "failed to get sleep pod", "", t)

	t.Run("Operator_test_sme_install", func(t *testing.T) {
		defer util.RecoverPanic(t)
		util.CheckPodRunning(meshNamespace, "app=wasm-cacher")

		log.Log.Info("Creating ServiceMeshExtension")
		util.KubeApplyContents("bookinfo", httpbinServiceMeshExtension)

		if err := checkSMEReady("bookinfo"); err != nil {
			t.Fatalf("error checking for SME header-append: %v", err)
		}

		time.Sleep(time.Duration(30) * time.Second)
		command := "curl -s -I httpbin:8000/headers"
		msg, err := util.PodExec("bookinfo", sleepPod, "sleep", command, false)
		if err != nil {
			t.Fatalf("error running command %q in pod %q: %v", command, sleepPod, err)
		}
		if !strings.Contains(msg, "maistra: rocks") {
			t.Fatalf("custom header not present: Expected value 'maistra: rocks'")
		}
	})
}

func checkSMEReady(n string) error {
	retry := util.Retrier{
		BaseDelay: 30 * time.Second,
		MaxDelay:  30 * time.Second,
		Retries:   6,
	}

	retryFn := func(_ context.Context, i int) error {
		ready, err := isSMEReady(n, "header-append")
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

func isSMEReady(ns, name string) (string, error) {
	res, err := util.Shell("kubectl -n %s get sme %s -o jsonpath='{.status.deployment.ready}'", ns, name)
	if err != nil {
		return "", fmt.Errorf("failed to get SME status for %s/%s: %v", ns, name, err)
	}
	return strings.Trim(res, "'"), nil
}

// Copyright 2021 Red Hat, Inc.
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

package authorizaton

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupTrustDomainMigration() {
	util.Log.Info("Cleanup")
	util.KubeDeleteContents("foo", TrustDomainPolicy)
	sleep := examples.Sleep{Namespace: "foo"}
	httpbin := examples.Httpbin{Namespace: "foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	sleep = examples.Sleep{Namespace: "bar"}
	sleep.Uninstall()
	applyTrustDomain("cluster.local", "", false)
}

func TestTrustDomainMigration(t *testing.T) {
	defer cleanupTrustDomainMigration()
	defer util.RecoverPanic(t)

	util.Log.Info("Trust Domain Migration")
	applyTrustDomain("old-td", "", true)

	// Deploy workloads
	httpbin := examples.Httpbin{Namespace: "foo"}
	util.Inspect(httpbin.Install(), "Failed to deploy httpbin", "", t)
	sleep := examples.Sleep{Namespace: "foo"}
	util.Inspect(sleep.Install(), "Failed to deploy sleep", "", t)
	sleep = examples.Sleep{Namespace: "bar"}
	util.Inspect(sleep.Install(), "Failed to deploy sleep", "", t)

	util.Log.Info("Apply deny all policy except sleep in bar namespace")
	util.KubeApplyContents("foo", TrustDomainPolicy)

	t.Run("Case 1: Verifying policy works", func(t *testing.T) {
		sleepPod, err := util.GetPodName("foo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		if err := checkOutput("foo", sleepPod, "sleep", cmd, "403"); err != nil {
			t.Fatal(err)
		}

		sleepPod, err = util.GetPodName("bar", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd = fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		if err := checkOutput("bar", sleepPod, "sleep", cmd, "200"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Case 2: Migrate trust domain without trust domain aliases", func(t *testing.T) {
		applyTrustDomain("new-td", "", true)

		// Restart workload pods
		util.Shell("oc -n foo delete pod --all")
		util.Shell("oc -n bar delete pod --all")
		util.Shell("oc -n foo wait --for condition=Ready --all pods --timeout 30s")
		util.Shell("oc -n bar wait --for condition=Ready --all pods --timeout 30s")

		// Both must return 403
		sleepPod, err := util.GetPodName("foo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		if err := checkOutput("foo", sleepPod, "sleep", cmd, "403"); err != nil {
			t.Fatal(err)
		}

		sleepPod, err = util.GetPodName("bar", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd = fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		if err := checkOutput("bar", sleepPod, "sleep", cmd, "403"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Case 3: Migrate trust domain with trust domain aliases", func(t *testing.T) {
		applyTrustDomain("new-td", "old-td", true)

		// Restart workload pods
		util.Shell("oc -n foo delete pod --all")
		util.Shell("oc -n bar delete pod --all")
		util.Shell("oc -n foo wait --for condition=Ready --all pods --timeout 30s")
		util.Shell("oc -n bar wait --for condition=Ready --all pods --timeout 30s")

		sleepPod, err := util.GetPodName("foo", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		if err := checkOutput("foo", sleepPod, "sleep", cmd, "403"); err != nil {
			t.Fatal(err)
		}

		// This must return 200, as in the first case
		sleepPod, err = util.GetPodName("bar", "app=sleep")
		util.Inspect(err, "Failed to get sleep pod name", "", t)
		cmd = fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
		if err := checkOutput("bar", sleepPod, "sleep", cmd, "200"); err != nil {
			t.Fatal(err)
		}
	})

}

func checkOutput(namespace, pod, container, cmd, expected string) error {
	retry := util.Retrier{
		BaseDelay: 5 * time.Second,
		MaxDelay:  10 * time.Second,
		Retries:   5,
	}

	util.Log.Infof("Verifying curl output, expecting %s", expected)

	retryFn := func(_ context.Context, i int) error {
		msg, err := util.PodExec(namespace, pod, container, cmd, true)
		if err != nil {
			return err
		}
		if !strings.Contains(msg, expected) {
			util.Log.Errorf("Attempt %d/%d - expected: %v; Got: %v", i, retry.Retries, expected, msg)
			return fmt.Errorf("expected: %v; Got: %v", expected, msg)
		}

		util.Log.Infof("Attempt %d/%d - Success, got %s", i, retry.Retries, msg)
		return nil
	}

	ctx := context.Background()
	_, err := retry.Retry(ctx, retryFn)
	return err
}

func applyTrustDomain(domain, alias string, mtls bool) {
	util.Log.Infof("Configuring  spec.security.trust.domain to %q and alias %q", domain, alias)

	if alias != "" {
		alias = fmt.Sprintf("%q", alias)
	}

	util.Shell(`oc -n istio-system patch smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":%v}, "trust":{"domain":"%s", "additionalDomains": [%s]}}}}'`, mtls, domain, alias)

	// Wait for the operator to reconcile the changes
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)

	// Restart istiod so it picks up the new trust domain
	util.Shell(`oc -n istio-system rollout restart deployment istiod-basic`)
	// wait 20 seconds and avoid a race condition checking wrong ingressgateway and istiod-basic pods
	time.Sleep(time.Duration(20) * time.Second)
	util.Shell(`oc -n istio-system wait --for condition=Ready --all pods --timeout 180s`)

	// Restart ingress gateway since we changed the mtls setting
	util.Shell(`oc -n istio-system rollout restart deployment istio-ingressgateway`)
	util.Shell(`oc -n istio-system wait --for condition=Ready --all pods --timeout 180s`)
}

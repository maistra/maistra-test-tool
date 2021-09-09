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
	sleep := examples.Sleep{"foo"}
	httpbin := examples.Httpbin{"foo"}
	sleep.Uninstall()
	httpbin.Uninstall()
	sleep = examples.Sleep{"bar"}
	sleep.Uninstall()
	util.Shell(`kubectl patch -n istio-system smcp/basic --type merge -p '{"spec":{"security":{"trust":{"domain":"cluster.local"}}}}'`)
	util.Shell(`kubectl patch -n istio-system smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":false},"controlPlane":{"mtls":false}}}}'`)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
	time.Sleep(time.Duration(40) * time.Second)
}

func TestTrustDomainMigration(t *testing.T) {
	defer cleanupTrustDomainMigration()
	defer util.RecoverPanic(t)

	util.Log.Info("Trust Domain Migration")
	util.Log.Info("Enable Control Plane MTLS")
	util.Shell(`kubectl patch -n istio-system smcp/basic --type merge -p '{"spec":{"security":{"dataPlane":{"mtls":true},"controlPlane":{"mtls":true}}}}'`)
	util.Log.Info("Configure  spec.security.trust.domain old-td")
	util.Shell(`kubectl patch -n istio-system smcp/basic --type merge -p '{"spec":{"security":{"trust":{"domain":"old-td"}}}}'`)
	util.Shell(`oc -n istio-system wait --for condition=Ready smcp/basic --timeout 180s`)
	time.Sleep(time.Duration(50) * time.Second)

	httpbin := examples.Httpbin{"foo"}
	httpbin.Install()
	sleep := examples.Sleep{"foo"}
	sleep.Install()
	sleep = examples.Sleep{"bar"}
	sleep.Install()

	util.Log.Info("Apply deny all policy except sleep in bar namespace")
	util.KubeApplyContents("foo", TrustDomainPolicy)
	time.Sleep(time.Duration(20) * time.Second)

	util.Log.Info("Verify setup")
	sleepPod, err := util.GetPodName("foo", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	cmd := fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
	msg, err := util.PodExec("foo", sleepPod, "sleep", cmd, true)
	util.Inspect(err, "Failed to get response", "", t)
	if !strings.Contains(msg, "403") {
		t.Errorf("Expected: 403; Got unexpected response code: %s", msg)
		util.Log.Errorf("Expected: 403; Got unexpected response code: %s", msg)
	} else {
		util.Log.Infof("Success. Get expected response: %s", msg)
	}

	sleepPod, err = util.GetPodName("bar", "app=sleep")
	util.Inspect(err, "Failed to get sleep pod name", "", t)
	cmd = fmt.Sprintf(`curl http://httpbin.%s:8000/ip -sS -o /dev/null -w "%%{http_code}\n"`, "foo")
	msg, err = util.PodExec("bar", sleepPod, "sleep", cmd, true)
	util.Inspect(err, "Failed to get response", "", t)
	if !strings.Contains(msg, "200") {
		t.Errorf("Expected: 200; Got unexpected response code: %s", msg)
		util.Log.Errorf("Expected: 200; Got unexpected response code: %s", msg)
	} else {
		util.Log.Infof("Success. Get expected response: %s", msg)
	}

}

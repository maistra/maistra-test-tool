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
	"istio.io/pkg/log"
	"maistra/util"
	"testing"
)

func testPilotLockIndefinitelyLock() {
	if _, err := util.Shell("oc rsh -n istio-system -c discovery $(oc get pods -n istio-system -l app=pilot --no-headers | awk '{print $1}') curl -v http://localhost:8080/debug/cdsz"); err != nil {
		t.Errorf("Pilot is Locked")
	}
}

func disableJaegerCollector() {
	util.Shell("oc rsh -n istio-system -c discovery $(oc get pods -n istio-system -l app=pilot --no-headers | awk '{print $1}') curl -v http://localhost:8080/debug/config_distribution?resource=policy/istio-system/disable-mtls-jaeger-collector")
}

func rolloutIngressGatewayIndefinitelyLock() {
	util.Shell("oc rollout -n istio-system restart deployment istio-ingressgateway")
}

func TestIndefinitelyLock(t *testing.T) {

	log.Info("Automation for MAISTRA-2101")

	testPilotLockIndefinitelyLock()
	go disableJaegerCollector()

	go rolloutIngressGatewayIndefinitelyLock()

	testPilotLockIndefinitelyLock()
}

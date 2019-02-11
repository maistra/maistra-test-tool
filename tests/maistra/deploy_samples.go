// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra

import (

	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)


func deployHttpbin(namespace, kubeconfig string) error {
	log.Info("# Deploy Httpbin")
	if err := util.KubeApply(namespace, httpbinYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(testNamespace, "app=httpbin", ""); err != nil {
		return err
	}
	return nil
}

func deployFortio(namespace, kubeconfig string) error {
	log.Info("# Deploy fortio")
	if err := util.KubeApply(namespace, httpbinFortioYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(testNamespace, "app=fortio", ""); err != nil {
		return err
	}
	return nil
}

func deployEcho(namespace, kubeconfig string) error {
	log.Info("# Deploy tcp-echo")
	if err := util.KubeApply(namespace, echoYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(testNamespace, "app=tcp-echo", ""); err != nil {
		return err
	}
	return nil
}

func deploySleep(namespace, kubeconfig string) error {
	log.Info("Deploy Sleep")
	if err := util.KubeApply(namespace, sleepYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(testNamespace, "app=sleep", ""); err != nil {
		return err
	}
	return nil
}

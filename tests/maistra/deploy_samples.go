// Copyright 2019 Red Hat, Inc.
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

package maistra

import (

	"time"

	"istio.io/istio/pkg/log"
	"istio.io/istio/tests/util"
)


func deployBookinfo(namespace, kubeconfig string, mtls bool) error {
	log.Info("# Deploy Bookinfo")
	if err := util.KubeApply(namespace, bookinfoYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=details", kubeconfig); err != nil {
		return err
	}
	if err := util.CheckPodRunning(namespace, "app=ratings", kubeconfig); err != nil {
		return err
	}
	if err := util.CheckPodRunning(namespace, "app=reviews", kubeconfig); err != nil {
		return err
	}
	if err := util.CheckPodRunning(namespace, "app=productpage", kubeconfig); err != nil {
		return err
	}

	// create gateway
	if err := util.KubeApply(namespace, bookinfoGateway, kubeconfig); err != nil {
		return err
	}
	// create destination rules
	if mtls {
		if err := util.KubeApply(namespace, bookinfoRuleAllTLSYaml, kubeconfig); err != nil {
			return err
		}
	} else {
		if err := util.KubeApply(namespace, bookinfoRuleAllYaml, kubeconfig); err != nil {
			return err
		}
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func cleanBookinfo(namespace, kubeconfig string) {
	log.Info("# Cleanup Bookinfo")
	util.KubeDelete(namespace, bookinfoRuleAllYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoGateway, kubeconfig)
	util.KubeDelete(namespace, bookinfoYaml, kubeconfig)
	log.Info("Waiting for rules to propagate. Sleep 30 seconds...")
	time.Sleep(time.Duration(30) * time.Second)
} 

func deployHttpbin(namespace, kubeconfig string) error {
	log.Info("# Deploy Httpbin")
	if err := util.KubeApply(namespace, httpbinYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=httpbin", kubeconfig); err != nil {
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
	if err := util.CheckPodRunning(namespace, "app=fortio", kubeconfig); err != nil {
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
	if err := util.CheckPodRunning(namespace, "app=tcp-echo", kubeconfig); err != nil {
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
	if err := util.CheckPodRunning(namespace, "app=sleep", kubeconfig); err != nil {
		return err
	}
	return nil
}

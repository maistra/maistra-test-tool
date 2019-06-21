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
	"maistra/util"
)

func deployBookinfo(namespace, kubeconfig string, mtls bool) error {
	log.Info("# Deploy Bookinfo")

	util.CreateNamespace(testNamespace, kubeconfig)
	util.OcGrantPermission("default", testNamespace, kubeconfig)

	if err := util.KubeApply(namespace, bookinfoYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	util.CheckPodRunning(namespace, "app=details", kubeconfig)
	util.CheckPodRunning(namespace, "app=ratings", kubeconfig)
	util.CheckPodRunning(namespace, "app=reviews,version=v1", kubeconfig)
	util.CheckPodRunning(namespace, "app=reviews,version=v2", kubeconfig)
	util.CheckPodRunning(namespace, "app=reviews,version=v3", kubeconfig)
	util.CheckPodRunning(namespace, "app=productpage", kubeconfig)

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
	log.Info("Waiting for rules to be cleaned up. Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
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
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
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
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func deployEcho(namespace, kubeconfig string) error {
	log.Info("# Deploy tcp-echo")
	if err := util.KubeApply(namespace, echoYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=tcp-echo,version=v1", kubeconfig); err != nil {
		return err
	}
	if err := util.CheckPodRunning(namespace, "app=tcp-echo,version=v2", kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
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
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func deployNginx(enableSidecar bool, namespace, kubeconfig string) error {
	log.Info("Deploy Nginx")
	if enableSidecar {
		if err := util.KubeApply(namespace, nginxYaml, kubeconfig); err != nil {
			return err
		}
	} else {
		if err := util.KubeApply(namespace, nginxNoSidecarYaml, kubeconfig); err != nil {
			return err
		}
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	if err := util.CheckPodRunning(namespace, "app=nginx", kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	return nil
}

func deployMongoDB(namespace, kubeconfig string) error {
	log.Info("Deploy MongoDB")
	if err := util.KubeApply(namespace, bookinfoDBYaml, kubeconfig); err != nil {
		return err
	}
	log.Info("Waiting for rules to propagate. Sleep 10 seconds...")
	time.Sleep(time.Duration(10) * time.Second)
	err := util.CheckPodRunning(namespace, "app=mongodb", kubeconfig)
	return err
}

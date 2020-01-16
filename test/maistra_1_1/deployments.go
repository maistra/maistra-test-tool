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

package main

import (
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func deployBookinfo(namespace string, mtls bool) {
	log.Info("* Deploying Bookinfo")

	util.KubeApply(namespace, bookinfoYaml, kubeconfig)
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(namespace, "app=details", kubeconfig)
	util.CheckPodRunning(namespace, "app=ratings", kubeconfig)
	util.CheckPodRunning(namespace, "app=reviews,version=v1", kubeconfig)
	util.CheckPodRunning(namespace, "app=reviews,version=v2", kubeconfig)
	util.CheckPodRunning(namespace, "app=reviews,version=v3", kubeconfig)
	util.CheckPodRunning(namespace, "app=productpage", kubeconfig)

	// Create gateway
	util.KubeApply(namespace, bookinfoGateway, kubeconfig)
	// Create destination rules all
	if mtls {
		util.KubeApply(namespace, bookinfoRuleAllTLSYaml, kubeconfig)
	} else {
		util.KubeApply(namespace, bookinfoRuleAllYaml, kubeconfig)
	}
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanBookinfo(namespace string) {
	log.Info("* Cleanup Bookinfo")

	util.KubeDelete(namespace, bookinfoRuleAllYaml, kubeconfig)
	util.KubeDelete(namespace, bookinfoGateway, kubeconfig)
	util.KubeDelete(namespace, bookinfoYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func deployHttpbin(namespace string) {
	log.Info("* Deploy Httpbin")

	util.KubeApply(namespace, httpbinYaml, kubeconfig)
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(namespace, "app=httpbin", kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanHttpbin(namespace string) {
	log.Info("* Cleanup Httpbin")

	util.KubeDelete(namespace, httpbinYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func deployFortio(namespace string) {
	log.Info("* Deploy fortio")

	util.KubeApply(namespace, httpbinFortioYaml, kubeconfig)
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(namespace, "app=fortio", kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanFortio(namespace string) {
	log.Info("* Cleanup fortio")

	util.KubeDelete(namespace, httpbinFortioYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func deployEcho(namespace string) {
	log.Info("* Deploy tcp-echo")

	util.KubeApply(namespace, echoYaml, kubeconfig)
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(namespace, "app=tcp-echo,version=v1", kubeconfig)
	util.CheckPodRunning(namespace, "app=tcp-echo,version=v2", kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanEcho(namespace string) {
	log.Info("* Cleanup tcp-echo")

	util.KubeDelete(namespace, echoYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func deploySleep(namespace string) {
	log.Info("* Deploy Sleep")

	util.KubeApply(namespace, sleepYaml, kubeconfig)
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(namespace, "app=sleep", kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanSleep(namespace string) {
	log.Info("* Cleanup Sleep")

	util.KubeDelete(namespace, sleepYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func deployNginx(enableSidecar bool, namespace string) {
	log.Info("* Deploy Nginx")

	if enableSidecar {
		util.KubeApply(namespace, nginxYaml, kubeconfig)
	} else {
		util.KubeApply(namespace, nginxNoSidecarYaml, kubeconfig)
	}
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime) * time.Second)
	util.CheckPodRunning(namespace, "app=nginx", kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanNginx(namespace string) {
	log.Info("* Cleanup Nginx")

	util.KubeDelete(namespace, nginxYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func deployMongoDB(namespace string) {
	log.Info("* Deploy MongoDB")

	util.KubeApply(namespace, bookinfoDBYaml, kubeconfig)
	log.Info("Waiting deployments complete...")
	time.Sleep(time.Duration(waitTime*6) * time.Second)
	util.CheckPodRunning(namespace, "app=mongodb", kubeconfig)
	time.Sleep(time.Duration(waitTime) * time.Second)
}

func cleanMongoDB(namespace string) {
	log.Info("* Cleanup MongoDB")

	util.KubeDelete(namespace, bookinfoDBYaml, kubeconfig)
	time.Sleep(time.Duration(waitTime*6) * time.Second)
}

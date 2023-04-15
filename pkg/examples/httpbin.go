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

package examples

import (
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

type Httpbin struct {
	Namespace string `json:"namespace,omitempty"`
}

func (h *Httpbin) Install() error {
	log.Log.Infof("Deploying Httpbin on namespace %s", h.Namespace)
	util.KubeApply(h.Namespace, httpbinYaml)
	_, err := util.CheckDeploymentIsReady(h.Namespace, "httpbin", time.Second*180)
	return err
}

func (h *Httpbin) InstallLegacy() {
	log.Log.Info("Deploy Httpbin")
	util.KubeApply(h.Namespace, httpbinLegacyYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallV1() {
	log.Log.Info("Deploy Httpbin-v1")
	util.KubeApply(h.Namespace, httpbinv1Yaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin,version=v1")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallV2() {
	log.Log.Info("Deploy Httpbin-v2")
	util.KubeApply(h.Namespace, httpbinv2Yaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin,version=v2")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) Uninstall() {
	log.Log.Infof("Removing Httpbin on namespace %s", h.Namespace)
	util.KubeDelete(h.Namespace, httpbinYaml)
	util.Shell(`oc -n %s wait --for=delete -l app=httpbin pods --timeout=30s`, h.Namespace)
}

func (h *Httpbin) UninstallV1() {
	log.Log.Info("Cleanup Httpbin-v1")
	util.KubeDelete(h.Namespace, httpbinv1Yaml)
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) UninstallV2() {
	log.Log.Info("Cleanup Httpbin-v2")
	util.KubeDelete(h.Namespace, httpbinv2Yaml)
	time.Sleep(time.Duration(10) * time.Second)
}

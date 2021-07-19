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
)

const (
	httpbinYaml       = "../samples/httpbin/httpbin.yaml"
	httpbinLegacyYaml = "../samples/httpbin/httpbin_legacy.yaml"
	httpbinv1Yaml     = "../samples/httpbin/httpbinv1.yaml"
	httpbinv2Yaml     = "../samples/httpbin/httpbinv2.yaml"
)

type Httpbin struct {
	Namespace string `json:"namespace,omitempty"`
}

func (h *Httpbin) Install() {
	util.Log.Info("Deploy Httpbin")
	util.KubeApply(h.Namespace, httpbinYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallLegacy() {
	util.Log.Info("Deploy Httpbin")
	util.KubeApply(h.Namespace, httpbinLegacyYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallV1() {
	util.Log.Info("Deploy Httpbin-v1")
	util.KubeApply(h.Namespace, httpbinv1Yaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin,version=v1")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallV2() {
	util.Log.Info("Deploy Httpbin-v2")
	util.KubeApply(h.Namespace, httpbinv2Yaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin,version=v2")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) Uninstall() {
	util.Log.Info("Cleanup Httpbin")
	util.KubeDelete(h.Namespace, httpbinYaml)
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) UninstallV1() {
	util.Log.Info("Cleanup Httpbin-v1")
	util.KubeDelete(h.Namespace, httpbinv1Yaml)
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) UninstallV2() {
	util.Log.Info("Cleanup Httpbin-v2")
	util.KubeDelete(h.Namespace, httpbinv2Yaml)
	time.Sleep(time.Duration(10) * time.Second)
}

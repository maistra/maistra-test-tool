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

type Echo struct {
	Namespace string `json:"namespace,omitempty"`
}

func (e *Echo) Install() {
	log.Log.Info("Deploy Echo")
	util.KubeApply(e.Namespace, echoYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(e.Namespace, "app=tcp-echo,version=v1")
	util.CheckPodRunning(e.Namespace, "app=tcp-echo,version=v2")
	time.Sleep(time.Duration(10) * time.Second)
}

func (e *Echo) InstallWithProxy() {
	log.Log.Info("Deploy Echo")
	util.KubeApply(e.Namespace, echoWithProxy)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(e.Namespace, "app=tcp-echo,version=v1")
	time.Sleep(time.Duration(10) * time.Second)
}

func (e *Echo) Uninstall() {
	log.Log.Info("Cleanup Echo")
	util.KubeDelete(e.Namespace, echoYaml)
	time.Sleep(time.Duration(10) * time.Second)
}

func (e *Echo) UninstallWithProxy() {
	log.Log.Info("Cleanup Echo")
	util.KubeDelete(e.Namespace, echoWithProxy)
	time.Sleep(time.Duration(10) * time.Second)
}

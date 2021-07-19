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
	sleepYaml       = "../samples/sleep/sleep.yaml"
	sleepLegacyYaml = "../samples/sleep/sleep_legacy.yaml"
)

type Sleep struct {
	Namespace string `json:"namespace,omitempty"`
}

func (s *Sleep) Install() {
	util.Log.Info("Deploy Sleep")
	util.KubeApply(s.Namespace, sleepYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(s.Namespace, "app=sleep")
	time.Sleep(time.Duration(10) * time.Second)
}

func (s *Sleep) InstallLegacy() {
	util.Log.Info("Deploy Sleep")
	util.KubeApply(s.Namespace, sleepLegacyYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(s.Namespace, "app=sleep")
	time.Sleep(time.Duration(10) * time.Second)
}

func (s *Sleep) Uninstall() {
	util.Log.Info("Cleanup Sleep")
	util.KubeDelete(s.Namespace, sleepYaml)
	time.Sleep(time.Duration(10) * time.Second)
}

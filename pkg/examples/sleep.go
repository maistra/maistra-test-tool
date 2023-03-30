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

// Define the sleep configmap file
const sleepConfigmap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: sleep-configmap
data:
  https-proxy: "{{ .HTTPProxy }}"
  http-proxy: "{{ .HTTPSProxy }}"
  no-proxy: "{{ .NoProxy }}"
`

var _ ExampleInterface = &Sleep{}

type Sleep struct {
	Namespace string `json:"namespace,omitempty"`
}

func (s *Sleep) Install() error {
	log.Log.Infof("Deploying Sleep in namespace %s", s.Namespace)
	proxy, _ := util.GetProxy()
	configmap := util.RunTemplate(sleepConfigmap, proxy)
	log.Log.Infof("Creating configmap %s", configmap)
	util.KubeApplyContents(s.Namespace, configmap)
	util.KubeApply(s.Namespace, sleepYaml)
	_, err := util.CheckDeploymentIsReady(s.Namespace, "sleep", time.Second*180)
	return err
}

func (s *Sleep) InstallLegacy() {
	log.Log.Info("Deploy Sleep")
	proxy, _ := util.GetProxy()
	configmap := util.RunTemplate(sleepConfigmap, proxy)
	log.Log.Infof("Creating configmap %s", configmap)
	util.KubeApplyContents(s.Namespace, configmap)
	util.KubeApply(s.Namespace, sleepLegacyYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(s.Namespace, "app=sleep")
	time.Sleep(time.Duration(10) * time.Second)
}

func (s *Sleep) Uninstall() {
	log.Log.Infof("Removing Sleep on namespace %s", s.Namespace)
	proxy, _ := util.GetProxy()
	configmap := util.RunTemplate(sleepConfigmap, proxy)
	util.KubeDeleteContents(s.Namespace, configmap)
	util.KubeDelete(s.Namespace, sleepYaml)
	util.Shell(`oc -n %s wait --for=delete -l app=sleep pods --timeout=30s`, s.Namespace)
}

func SleepConfigMap() string {
	return sleepConfigmap
}

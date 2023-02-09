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
	"os"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

var _ ExampleInterface = &Httpbin{}

var ipv6 string = util.Getenv("IPV6", "false")

type Httpbin struct {
	Namespace string `json:"namespace,omitempty"`
}

func (h *Httpbin) Install() error {
	util.Log.Infof("Deploying Httpbin on namespace %s", h.Namespace)
	template, err := GetHttpbinTemplate(httpbinYaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
		return err
	}
	util.Log.Infof("Deploying Httpbin on namespace %s", h.Namespace)
	util.KubeApplyContents(h.Namespace, template)
	_, err = util.CheckDeploymentIsReady(h.Namespace, "httpbin", time.Second*180)
	return err
}

func (h *Httpbin) InstallLegacy() {
	util.Log.Info("Deploy Httpbin Legacy")
	template, err := GetHttpbinTemplate(httpbinLegacyYaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
	}
	util.Log.Infof("Deploying httpbinLegacyYaml on namespace %s", h.Namespace)
	util.KubeApplyContents(h.Namespace, template)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallV1() {
	util.Log.Info("Deploy Httpbin-v1")
	template, err := GetHttpbinTemplate(httpbinv1Yaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
	}
	util.Log.Infof("Deploying httpbinv1Yaml on namespace %s", h.Namespace)
	util.KubeApplyContents(h.Namespace, template)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin,version=v1")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) InstallV2() {
	util.Log.Info("Deploy Httpbin-v2")
	template, err := GetHttpbinTemplate(httpbinv2Yaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
	}
	util.Log.Infof("Deploying httpbinv2Yaml on namespace %s", h.Namespace)
	util.KubeApplyContents(h.Namespace, template)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(h.Namespace, "app=httpbin,version=v2")
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) Uninstall() {
	util.Log.Infof("Removing Httpbin on namespace %s", h.Namespace)
	template, err := GetHttpbinTemplate(httpbinYaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
	}
	util.KubeDelete(h.Namespace, template)
	util.Shell(`oc -n %s wait --for=delete -l app=httpbin pods --timeout=30s`, h.Namespace)
}

func (h *Httpbin) UninstallV1() {
	util.Log.Info("Cleanup Httpbin-v1")
	template, err := GetHttpbinTemplate(httpbinv1Yaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
	}
	util.KubeDelete(h.Namespace, template)
	time.Sleep(time.Duration(10) * time.Second)
}

func (h *Httpbin) UninstallV2() {
	util.Log.Info("Cleanup Httpbin-v2")
	template, err := GetHttpbinTemplate(httpbinv2Yaml)
	if err != nil {
		util.Log.Errorf("Failed to get Httpbin template: %s", err)
	}
	util.KubeDelete(h.Namespace, template)
	time.Sleep(time.Duration(10) * time.Second)
}

// Get the template file and fill in the values for httpbin yaml file
func GetHttpbinTemplate(templateFile string) (string, error) {
	type bindAddress struct {
		Bind string `default:"0.0.0.0:8000"`
	}
	values := bindAddress{}
	templateBytes, err := os.ReadFile(templateFile)
	templateString := string(templateBytes)
	if err != nil {
		util.Log.Errorf("Failed to read httpbin yaml file: %s", err)
		return "", err
	}
	if ipv6 == "true" {
		util.Log.Info("Deploy Httpbin to bind IPV6 address [::]:8000")
		values.Bind = "[::]:8000"
	} else {
		util.Log.Info("Deploy Httpbin to bind IPV4 address 0.0.0.0:8000")
	}
	template := util.RunTemplate(templateString, values)
	return template, nil
}

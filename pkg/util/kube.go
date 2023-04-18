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

package util

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

const (
	podFailedGet = "Failed_Get"
	// The index of STATUS field in kubectl CLI output.
	statusField = 2
)

type Proxy struct {
	HTTPProxy  string `json:"httpProxy"`
	HTTPSProxy string `json:"httpsProxy"`
	NoProxy    string `json:"noProxy"`
}

// PodInfo contains pod's information such as name and IP address
type PodInfo struct {
	// Name is the pod's name
	Name string
	// IPAddr is the pod's IP
	IPAddr string
}

// KubeApplyContents kubectl apply from contents
func KubeApplyContents(namespace, yamlContents string) error {
	_, err := ShellWithInput(yamlContents, kubeCommand("apply", namespace, "-"))
	return err
}

func kubeCommand(subCommand, namespace, yamlFileName string) string {
	if namespace == "" {
		return fmt.Sprintf("kubectl %s -f %s", subCommand, yamlFileName)
	}
	return fmt.Sprintf("kubectl %s -n %s -f %s", subCommand, namespace, yamlFileName)
}

// KubeDeleteContents kubectl apply from contents
func KubeDeleteContents(namespace, yamlContents string) error {
	tmpfile, err := WriteTempfile(os.TempDir(), "kubedelete", ".yaml", yamlContents)
	if err != nil {
		return err
	}
	defer removeFile(tmpfile)
	return KubeDelete(namespace, tmpfile)
}

func removeFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Log.Errorf("Unable to remove %s: %v", path, err)
	}
}

// KubeDelete kubectl delete from file
func KubeDelete(namespace, yamlFileName string) error {
	_, err := ShellMuteOutputError(kubeCommand("delete", namespace, yamlFileName) + " --ignore-not-found")
	return err
}

// GetPodNames gets names of all pods in specific namespace and return in a slice
func GetPodNames(n string) (pods []string) {
	res, err := Shell("kubectl -n %s get pods -o jsonpath='{.items[*].metadata.name}'", n)
	if err != nil {
		log.Log.Infof("Failed to get pods name in namespace %s: %s", n, err)
		return
	}
	res = strings.Trim(res, "'")
	pods = strings.Split(res, " ")
	log.Log.Infof("Existing pods: %v", pods)
	return
}

// GetPodStatus gets status of a pod from a namespace
// Note: It is not enough to check pod phase, which only implies there is at
// least one container running. Use kubectl CLI to get status so that we can
// ensure that all containers are running.
func GetPodStatus(n, pod string) string {
	status, err := Shell("kubectl -n %s get pods %s --no-headers", n, pod)
	if err != nil {
		log.Log.Infof("Failed to get status of pod %s in namespace %s: %s", pod, n, err)
		status = podFailedGet
	}
	f := strings.Fields(status)
	if len(f) > statusField {
		return f[statusField]
	}
	return ""
}

// GetPodName gets the pod name for the given namespace and label selector
func GetPodName(ns, labelSelector string) (pod string, err error) {
	pod, err = Shell("kubectl -n %s get pod -l %s -o jsonpath='{.items[0].metadata.name}'", ns, labelSelector)
	if err != nil {
		log.Log.Warnf("could not get %s pod: %v", labelSelector, err)
		return
	}
	pod = strings.Trim(pod, "'")
	log.Log.Infof("%s pod name: %s", labelSelector, pod)
	return
}

// CreateTLSSecret creates a secret from the provided cert and key files
func CreateTLSSecret(secretName, ns, keyFile, certFile string) (string, error) {
	// cmd := fmt.Sprintf("kubectl create secret tls %s -n %s --key %s --cert %s", secretName, n, keyFile, certFile)
	// return Shell(cmd)
	return Shell("kubectl create secret tls %s -n %s --key %s --cert %s", secretName, ns, keyFile, certFile)
}

// CheckPodRunning return if a given pod with labeled name in a namespace are in "Running" status
func CheckPodRunning(n, name string) error {
	retry := Retrier{
		BaseDelay: 30 * time.Second,
		MaxDelay:  30 * time.Second,
		Retries:   6,
	}

	retryFn := func(_ context.Context, i int) error {
		pod, err := GetPodName(n, name)
		if err != nil {
			return err
		}
		ready := true
		if status := GetPodStatus(n, pod); status != "Running" {
			log.Log.Infof("%s in namespace %s is not running: %s", pod, n, status)
			ready = false
		}

		if !ready {
			return fmt.Errorf("pod %s is not ready", pod)
		}
		return nil
	}
	ctx := context.Background()
	_, err := retry.Retry(ctx, retryFn)
	if err != nil {
		return err
	}
	log.Log.Infof("Got the pod name=%s running!", name)
	return nil
}

// GetJsonObject get json string as input and returns a map of the json string
func GetJsonObject(jsonString string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetProxy returns the Proxy struc object from the cluster
func GetProxy() (*Proxy, error) {
	Proxy := &Proxy{}
	proxyString, err := ShellSilent(`oc get Proxy -o json`)
	if err != nil {
		log.Log.Error("Error getting proxy String")
		return nil, err
	}
	proxyObject, err := GetJsonObject(proxyString)
	if err != nil {
		log.Log.Error("Error getting proxy object")
		return nil, err
	}
	proxyObject = proxyObject["items"].([]interface{})[0].(map[string]interface{})
	if proxyObject["status"] != nil {
		proxyStatus := proxyObject["status"].(map[string]interface{})
		if proxyStatus["httpProxy"] != nil && proxyStatus["httpsProxy"] != nil {
			Proxy.HTTPProxy = proxyStatus["httpProxy"].(string)
			Proxy.HTTPSProxy = proxyStatus["httpsProxy"].(string)
			log.Log.Info("Current httpProxy: ", Proxy.HTTPProxy)
			log.Log.Info("Current httpsProxy: ", Proxy.HTTPSProxy)
			if proxyStatus["noProxy"] != nil {
				Proxy.NoProxy = proxyStatus["noProxy"].(string)
				log.Log.Info("Current noProxy: ", Proxy.NoProxy)
			}
		}
	} else {
		log.Log.Info("No proxy variables need to be configured")
		Proxy.HTTPProxy = ""
		Proxy.HTTPSProxy = ""
		Proxy.NoProxy = ""
	}
	return Proxy, nil
}

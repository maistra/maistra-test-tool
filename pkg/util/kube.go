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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

const (
	podFailedGet = "Failed_Get"
	// The index of STATUS field in kubectl CLI output.
	statusField          = 2
	defaultClusterSubnet = "24"

	// NodePortServiceType NodePort type of Kubernetes Service
	NodePortServiceType = "NodePort"

	// LoadBalancerServiceType LoadBalancer type of Kubernetes Service
	LoadBalancerServiceType = "LoadBalancer"
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

// Fill complete a template with given values and generate a new output file
func Fill(outFile, inFile string, values interface{}) error {
	tmpl, err := template.ParseFiles(inFile)
	if err != nil {
		return err
	}

	var filled bytes.Buffer
	w := bufio.NewWriter(&filled)
	if err := tmpl.Execute(w, values); err != nil {
		return err
	}

	if err := w.Flush(); err != nil {
		return err
	}

	if err := ioutil.WriteFile(outFile, filled.Bytes(), 0644); err != nil {
		return err
	}
	log.Log.Infof("Created %s from template %s", outFile, inFile)
	return nil
}

// CreateAndFill fills in the given yaml template with the values and generates a temp file for the completed yaml.
func CreateAndFill(outDir, templateFile string, values interface{}) (string, error) {
	outFile, err := CreateTempfile(outDir, filepath.Base(templateFile), "yaml")
	if err != nil {
		log.Log.Errorf("Failed to generate yaml %s: %v", templateFile, err)
		return "", err
	}
	if err := Fill(outFile, templateFile, values); err != nil {
		log.Log.Errorf("Failed to generate yaml for template %s: %v", templateFile, err)
		return "", err
	}
	return outFile, nil
}

// DeleteNamespace delete a kubernetes namespace
func DeleteNamespace(n string) error {
	if _, err := Shell("kubectl delete project %s", n); err != nil {
		if !strings.Contains(err.Error(), "not found") {
			return err
		}
	}
	log.Log.Infof("namespace %s deleted\n", n)
	return nil
}

// DeleteDeployment deletes deployment from the specified namespace
func DeleteDeployment(d string, n string) error {
	_, err := Shell("kubectl delete deployment %s -n %s", d, n)
	return err
}

// NamespaceDeleted check if a kubernete namespace is deleted
func NamespaceDeleted(n string) (bool, error) {
	output, err := ShellSilent("kubectl get namespace %s -o name", n)
	if strings.Contains(output, "NotFound") {
		return true, nil
	}
	return false, err
}

// ValidatingWebhookConfigurationExists check if a kubernetes ValidatingWebhookConfiguration is deleted
func ValidatingWebhookConfigurationExists(name string) bool {
	output, _ := ShellSilent("kubectl get validatingwebhookconfiguration %s -o name", name)
	return !strings.Contains(output, "NotFound")
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

// KubeApply kubectl apply from file
func KubeApply(namespace, yamlFileName string) error {
	_, err := Shell(kubeCommand("apply", namespace, yamlFileName))
	return err
}

// KubeGetYaml kubectl get yaml content for given resource.
func KubeGetYaml(namespace, resource, name string) (string, error) {
	if namespace == "" {
		namespace = "default"
	}
	cmd := fmt.Sprintf("kubectl get %s %s -n %s -o yaml --export", resource, name, namespace)

	return Shell(cmd)
}

// KubeApplyContentSilent kubectl apply from contents silently
func KubeApplyContentSilent(namespace, yamlContents string) error {
	tmpfile, err := WriteTempfile(os.TempDir(), "kubeapply", ".yaml", yamlContents)
	if err != nil {
		return err
	}
	defer removeFile(tmpfile)
	return KubeApplySilent(namespace, tmpfile)
}

// KubeApplySilent kubectl apply from file silently
func KubeApplySilent(namespace, yamlFileName string) error {
	_, err := ShellSilent(kubeCommand("apply", namespace, yamlFileName))
	return err
}

// KubeScale kubectl scale a pod specified using typeName
func KubeScale(namespace, typeName string, replicaCount int) error {
	kubecommand := fmt.Sprintf("kubectl scale -n %s --replicas=%d %s", namespace, replicaCount, typeName)
	_, err := Shell(kubecommand)
	return err
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

// GetKubeMasterIP returns the IP address of the kubernetes master service.
func GetKubeMasterIP() (string, error) {
	return ShellSilent("kubectl get svc kubernetes -n default -o jsonpath='{.spec.clusterIP}'")
}

// GetClusterSubnet returns the subnet (in CIDR form, e.g. "24") for the nodes in the cluster.
func GetClusterSubnet() (string, error) {
	cidr, err := ShellSilent("kubectl get nodes -o jsonpath='{.items[0].spec.podCIDR}'")
	if err != nil {
		// This command should never fail. If the field isn'T found, it will just return and empty string.
		return "", err
	}
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		// TODO(nmittler): Need a way to get the subnet on minikube. For now, just return a default value.
		log.Log.Info("unable to identify cluster subnet. running on minikube?")
		return defaultClusterSubnet, nil
	}
	return parts[1], nil
}

// GetPodAnnotations gets a map annotations from a pod name for the given: namespace, pod label and retry times for checking the pod annotations
func GetPodAnnotations(n, podName string, retries int) (map[string]string, error) {
	retry := Retrier{
		BaseDelay: 1 * time.Second,
		MaxDelay:  1 * time.Second,
		Retries:   retries,
	}
	var annotations map[string]string
	_, err := retry.Retry(context.Background(), func(_ context.Context, _ int) error {
		output, err := Shell("kubectl get pod %s -n %s -o jsonpath='{.metadata.annotations}'", podName, n)
		if err != nil {
			log.Log.Infof("Failed to get pods by label %s in namespace %s: %s", podName, n, err)
			return fmt.Errorf("failed to get pod %s: %s", podName, err)
		}
		if output != "" {
			if err := json.Unmarshal([]byte(output), &annotations); err != nil {
				return fmt.Errorf("failed to unmarshal pod annotations: %s", err)
			}
			return nil
		}
		return fmt.Errorf("pod annotations not found yet")
	})
	return annotations, err
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

// Check PodReady (Example: 2/2 container running on a pod) returns true if the pod is ready. Params: namespace, pod label and timeout to check ready pod.
func CheckPodReady(ns, selector string, retries int) (bool, error) {
	ready := false
	retry := Retrier{
		BaseDelay: 10 * time.Second,
		MaxDelay:  10 * time.Second,
		Retries:   retries,
	}
	_, err := retry.Retry(context.Background(), func(_ context.Context, _ int) error {
		output, err := Shell(`oc get pods -n istio-system -l %s -o jsonpath='{range .items[*]}{.status.containerStatuses[*].ready}'`, selector)
		if err != nil {
			return fmt.Errorf("failed to get pod: %s", err)
		}
		if strings.Contains(output, "false") {
			return fmt.Errorf("pod is not ready")
		} else {
			ready = true
			return nil
		}
	})
	return ready, err
}

// CheckPodDeletion returns true if the pod is deleted. Params: label of the pod, the Pod Name to check,  namespace and a timeout
func CheckPodDeletion(n, labelSelector string, previousPodName string, timeout int) (deleted bool, err error) {
	deleted = false
	for i := 0; i < timeout; i++ {
		pod, err := GetPodName(n, labelSelector)
		if err != nil {
			deleted = true
			return deleted, err
		}
		if pod != previousPodName {
			deleted = true
			return deleted, nil
		}
		if pod == "" {
			deleted = true
			return deleted, nil
		}
		time.Sleep(time.Duration(1) * time.Second)
	}
	return deleted, nil
}

// GetPodLogsForLabel gets the logs for the given label selector and container
func GetPodLogsForLabel(n, labelSelector string, container string, tail, alsoShowPreviousPodLogs bool) string {
	pod, err := GetPodName(n, labelSelector)
	if err != nil {
		return ""
	}
	return GetPodLogs(n, pod, container, tail, alsoShowPreviousPodLogs)
}

// GetPodLogs retrieves the logs for the given namespace, pod and container.
func GetPodLogs(n, pod, container string, tail, alsoShowPreviousPodLogs bool) string {
	tailOption := ""
	if tail {
		tailOption = "--tail=40"
	}
	o1 := ""
	if alsoShowPreviousPodLogs {
		log.Log.Info("Expect and ignore an error getting crash logs when there are no crash (-p invocation)")
		// Do not use Shell. It dumps the entire log on the console and makes the test unusable due to very large amount of output
		o1, _ = ShellMuteOutput("kubectl --namespace %s logs %s -c %s %s -p", n, pod, container, tailOption)
		o1 += "\n"
	}
	// Do not use Shell. It dumps the entire log on the console and makes the test unusable due to very large amount of output
	o2, _ := ShellMuteOutput("kubectl --namespace %s logs %s -c %s %s", n, pod, container, tailOption)
	return o1 + o2
}

// GetConfigs retrieves the configurations for the list of resources.
func GetConfigs(names ...string) (string, error) {
	cmd := fmt.Sprintf("kubectl get %s --all-namespaces -o yaml", strings.Join(names, ","))
	return Shell(cmd)
}

// PodExec runs the specified command on the container for the specified namespace and pod
func PodExec(n, pod, container, command string, muteOutput bool) (string, error) {
	if muteOutput {
		return ShellSilent("kubectl exec %s -n %s -c %s -- %s", pod, n, container, command)
	}
	return Shell("kubectl exec %s -n %s -c %s -- %s ", pod, n, container, command)
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

// CreateMultiClusterSecret will create the secret associated with the remote cluster
func CreateMultiClusterSecret(namespace string, remoteKubeConfig string, localKubeConfig string) error {
	const (
		secretLabel = "istio/multiCluster"
		labelValue  = "true"
	)
	secretName := filepath.Base(remoteKubeConfig)

	_, err := ShellMuteOutput("kubectl create secret generic %s --from-file %s -n %s --kubeconfig=%s", secretName, remoteKubeConfig, namespace, localKubeConfig)
	if err != nil {
		log.Log.Infof("Failed to create secret %s\n", secretName)
		return err
	}
	log.Log.Infof("Secret %s created\n", secretName)

	// label the secret for use as istio/multiCluster config
	_, err = ShellMuteOutput("kubectl label secret %s %s=%s -n %s --kubeconfig=%s",
		secretName, secretLabel, labelValue, namespace, localKubeConfig)
	if err != nil {
		return err
	}

	log.Log.Infof("Secret %s labeled with %s=%s\n", secretName, secretLabel, labelValue)
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

// GetHTTPProxy returns the Proxy struc object from the cluster
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

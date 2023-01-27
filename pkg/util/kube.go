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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
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

var (
	logDumpResources = []string{
		"pod",
		"service",
		"ingress",
	}
)

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
	Log.Infof("Created %s from template %s", outFile, inFile)
	return nil
}

// CreateAndFill fills in the given yaml template with the values and generates a temp file for the completed yaml.
func CreateAndFill(outDir, templateFile string, values interface{}) (string, error) {
	outFile, err := CreateTempfile(outDir, filepath.Base(templateFile), "yaml")
	if err != nil {
		Log.Errorf("Failed to generate yaml %s: %v", templateFile, err)
		return "", err
	}
	if err := Fill(outFile, templateFile, values); err != nil {
		Log.Errorf("Failed to generate yaml for template %s: %v", templateFile, err)
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
	Log.Infof("namespace %s deleted\n", n)
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
	tmpfile, err := WriteTempfile(os.TempDir(), "kubeapply", ".yaml", yamlContents)
	if err != nil {
		return err
	}
	defer removeFile(tmpfile)
	return KubeApply(namespace, tmpfile)
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
		Log.Errorf("Unable to remove %s: %v", path, err)
	}
}

// KubeDelete kubectl delete from file
func KubeDelete(namespace, yamlFileName string) error {
	_, err := ShellMuteOutputError(kubeCommand("delete", namespace, yamlFileName))
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
		// This command should never fail. If the field isn't found, it will just return and empty string.
		return "", err
	}
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		// TODO(nmittler): Need a way to get the subnet on minikube. For now, just return a default value.
		Log.Info("unable to identify cluster subnet. running on minikube?")
		return defaultClusterSubnet, nil
	}
	return parts[1], nil
}

func getRetrier(serviceType string) Retrier {
	baseDelay := 1 * time.Second
	maxDelay := 1 * time.Second
	retries := 300 // ~5 minutes

	if serviceType == NodePortServiceType {
		baseDelay = 5 * time.Second
		maxDelay = 5 * time.Second
		retries = 20
	}

	return Retrier{
		BaseDelay: baseDelay,
		MaxDelay:  maxDelay,
		Retries:   retries,
	}
}

func getServiceLoadBalancer(name, namespace string) (string, error) {
	ip, err := ShellSilent(
		"kubectl get svc %s -n %s -o jsonpath='{.status.loadBalancer.ingress[*].ip}'",
		name, namespace)

	if err != nil {
		return "", err
	}

	ip = strings.Trim(ip, "'")
	ri := regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`)
	if ri.FindString(ip) == "" {
		return "", errors.New("ingress ip not available yet")
	}

	return ip, nil
}

func getServiceNodePort(serviceName, podLabel, namespace string) (string, error) {
	ip, err := Shell(
		"kubectl get po -l istio=%s -n %s -o jsonpath='{.items[0].status.hostIP}'",
		podLabel, namespace)

	if err != nil {
		return "", err
	}

	ip = strings.Trim(ip, "'")
	ri := regexp.MustCompile(`^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$`)
	if ri.FindString(ip) == "" {
		return "", fmt.Errorf("the ip of %s is not available yet", serviceName)
	}

	port, err := getServicePort(serviceName, namespace)
	if err != nil {
		return "", err
	}

	return ip + ":" + port, nil
}

func getServicePort(serviceName, namespace string) (string, error) {
	port, err := Shell(
		"kubectl get svc %s -n %s -o jsonpath='{.spec.ports[0].nodePort}'",
		serviceName, namespace)

	if err != nil {
		return "", err
	}

	port = strings.Trim(port, "'")
	rp := regexp.MustCompile(`^[0-9]{1,5}$`)
	if rp.FindString(port) == "" {
		err = fmt.Errorf("unable to find the port of %s", serviceName)
		Log.Warn(err)
		return "", err
	}
	return port, nil
}

// GetIngressPodNames get the pod names for the Istio ingress deployment.
func GetIngressPodNames(n string) ([]string, error) {
	res, err := Shell("kubectl get pod -l istio=ingress -n %s -o jsonpath='{.items[*].metadata.name}'", n)
	if err != nil {
		return nil, err
	}
	res = strings.Trim(res, "'")
	return strings.Split(res, " "), nil
}

// GetAppPodsInfo returns a map of a list of PodInfo
func GetAppPodsInfo(n string, label string) ([]string, map[string][]string, error) {
	// This will return a table where c0=pod_name and c1=label_value and c2=IPAddr.
	// The columns are separated by a space and each result is on a separate line (separated by '\n').
	res, err := Shell("kubectl -n %s -l=%s get pods -o=jsonpath='{range .items[*]}{.metadata.name}{\" \"}{"+
		".metadata.labels.%s}{\" \"}{.status.podIP}{\"\\n\"}{end}'", n, label, label)
	if err != nil {
		Log.Infof("Failed to get pods by label %s in namespace %s: %s", label, n, err)
		return nil, nil, err
	}

	var podNames []string
	eps := make(map[string][]string)
	for _, line := range strings.Split(res, "\n") {
		f := strings.Fields(line)
		if len(f) >= 3 {
			podNames = append(podNames, f[0])
			eps[f[1]] = append(eps[f[1]], f[2])
		}
	}

	return podNames, eps, nil
}

// GetAppPods gets a map of app names to the pods for the app, for the given namespace
func GetAppPods(n string) (map[string][]string, error) {
	podLabels, err := GetPodLabelValues(n, "app")
	if err != nil {
		return nil, err
	}

	m := make(map[string][]string)
	for podName, app := range podLabels {
		m[app] = append(m[app], podName)
	}
	return m, nil
}

// GetPodLabelValues gets a map of pod name to label value for the given label and namespace
func GetPodLabelValues(n, label string) (map[string]string, error) {
	// This will return a table where c0=pod_name and c1=label_value.
	// The columns are separated by a space and each result is on a separate line (separated by '\n').
	res, err := Shell("kubectl -n %s -l=%s get pods -o=jsonpath='{range .items[*]}{.metadata.name}{\" \"}{"+
		".metadata.labels.%s}{\"\\n\"}{end}'", n, label, label)
	if err != nil {
		Log.Infof("Failed to get pods by label %s in namespace %s: %s", label, n, err)
		return nil, err
	}

	// Split the lines in the result
	m := make(map[string]string)
	for _, line := range strings.Split(res, "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 {
			m[f[0]] = f[1]
		}
	}

	return m, nil
}

// GetPodAnnotations gets a map annotations from a pod name for the given: namespace, pod label and a timeout for checking the pod annotations
func GetPodAnnotations(n, podName string, timeout int) (map[string]string, error) {
	retry := Retrier{
		BaseDelay: 1 * time.Second,
		MaxDelay:  1 * time.Second,
		Retries:   timeout,
	}
	var annotations map[string]string
	_, err := retry.Retry(context.Background(), func(_ context.Context, _ int) error {
		output, error := Shell("kubectl get pod %s -n %s -o jsonpath='{.metadata.annotations}'", podName, n)
		if error != nil {
			Log.Infof("Failed to get pods by label %s in namespace %s: %s", podName, n, error)
			return &Break{error}
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
		Log.Infof("Failed to get pods name in namespace %s: %s", n, err)
		return
	}
	res = strings.Trim(res, "'")
	pods = strings.Split(res, " ")
	Log.Infof("Existing pods: %v", pods)
	return
}

// GetPodStatus gets status of a pod from a namespace
// Note: It is not enough to check pod phase, which only implies there is at
// least one container running. Use kubectl CLI to get status so that we can
// ensure that all containers are running.
func GetPodStatus(n, pod string) string {
	status, err := Shell("kubectl -n %s get pods %s --no-headers", n, pod)
	if err != nil {
		Log.Infof("Failed to get status of pod %s in namespace %s: %s", pod, n, err)
		status = podFailedGet
	}
	f := strings.Fields(status)
	if len(f) > statusField {
		return f[statusField]
	}
	return ""
}

// GetPodName gets the pod name for the given namespace and label selector
func GetPodName(n, labelSelector string) (pod string, err error) {
	pod, err = Shell("kubectl -n %s get pod -l %s -o jsonpath='{.items[0].metadata.name}'", n, labelSelector)
	if err != nil {
		Log.Warnf("could not get %s pod: %v", labelSelector, err)
		return
	}
	pod = strings.Trim(pod, "'")
	Log.Infof("%s pod name: %s", labelSelector, pod)
	return
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
		Log.Info("Expect and ignore an error getting crash logs when there are no crash (-p invocation)")
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
func CreateTLSSecret(secretName, n, keyFile, certFile string) (string, error) {
	//cmd := fmt.Sprintf("kubectl create secret tls %s -n %s --key %s --cert %s", secretName, n, keyFile, certFile)
	//return Shell(cmd)
	return Shell("kubectl create secret tls %s -n %s --key %s --cert %s", secretName, n, keyFile, certFile)
}

// CheckPodsRunningWithMaxDuration returns if all pods in a namespace are in "Running" status
// Also check container status to be running.
func CheckPodsRunningWithMaxDuration(n string, maxDuration time.Duration) (ready bool) {
	if err := WaitForDeploymentsReady(n, maxDuration); err != nil {
		Log.Errorf("CheckPodsRunning: %v", err.Error())
		return false
	}

	return true
}

// CheckPodsRunning returns readiness of all pods within a namespace. It will wait for upto 2 mins.
// use WithMaxDuration to specify a duration.
func CheckPodsRunning(n string) (ready bool) {
	return CheckPodsRunningWithMaxDuration(n, 2*time.Minute)
}

// CheckDeployment gets status of a deployment from a namespace
func CheckDeployment(ctx context.Context, namespace, deployment string) error {
	if deployment == "deployments/istio-sidecar-injector" {
		// This can be deployed by previous tests, but doesn't complete currently, blocking the test.
		return nil
	}
	errc := make(chan error)
	go func() {
		if _, err := ShellMuteOutput("kubectl -n %s rollout status %s", namespace, deployment); err != nil {
			errc <- fmt.Errorf("%s in namespace %s failed", deployment, namespace)
		}
		errc <- nil
	}()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CheckDeploymentRemoved waits until a deployment is removed or times out
func CheckDeploymentRemoved(namespace, deployment string) error {
	retry := Retrier{
		BaseDelay: 5 * time.Second,
		MaxDelay:  5 * time.Second,
		Retries:   60,
	}

	pod, err := GetPodName(namespace, "name="+deployment)
	// Pod has been removed
	if err != nil {
		Log.Infof("pod %s is successfully removed", pod)
		return nil
	}
	retryFn := func(_ context.Context, i int) error {
		_, err := Shell("kubectl get pods %s -n %s", pod, namespace)
		if err != nil {
			Log.Infof("pod %s is successfully removed", pod)
			return nil
		}
		return fmt.Errorf("%s in namespace %s still exists", pod, namespace)
	}
	ctx := context.Background()
	_, err = retry.Retry(ctx, retryFn)
	if err != nil {
		return err
	}
	return nil
}

// WaitForDeploymentsReady wait up to 'timeout' duration
// return an error if deployments are not ready
func WaitForDeploymentsReady(ns string, timeout time.Duration) error {
	retry := Retrier{
		BaseDelay:   10 * time.Second,
		MaxDelay:    10 * time.Second,
		MaxDuration: timeout,
		Retries:     20,
	}

	_, err := retry.Retry(context.Background(), func(_ context.Context, _ int) error {
		nr, err := CheckDeploymentsReady(ns)
		if err != nil {
			return &Break{err}
		}

		if nr == 0 { // done
			return nil
		}
		return fmt.Errorf("%d deployments not ready", nr)
	})
	return err
}

// CheckDeploymentsReady checks if deployment resources are ready.
// get podsReady() sometimes gets pods created by the "Job" resource which never reach the "Running" steady state.
func CheckDeploymentsReady(ns string) (int, error) {
	CMD := "kubectl -n %s get deployments -o jsonpath='{range .items[*]}{@.metadata.name}{\" \"}" +
		"{@.status.availableReplicas}{\"\\n\"}{end}'"
	out, err := Shell(fmt.Sprintf(CMD, ns))

	if err != nil {
		return 0, fmt.Errorf("could not list deployments in namespace %q: %v", ns, err)
	}

	notReady := 0
	for _, line := range strings.Split(out, "\n") {
		flds := strings.Fields(line)
		if len(flds) < 2 {
			continue
		}
		if flds[1] == "0" { // no replicas ready
			notReady++
		}
	}

	if notReady == 0 {
		Log.Infof("All deployments are ready")
	}
	return notReady, nil
}

// GetKubeConfig will create a kubeconfig file based on the active environment the test is run in
func GetKubeConfig(filename string) error {
	_, err := ShellMuteOutput("kubectl config view --raw=true --minify=true > %s", filename)
	if err != nil {
		return err
	}
	Log.Infof("kubeconfig file %s created\n", filename)
	return nil
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
			Log.Infof("%s in namespace %s is not running: %s", pod, n, status)
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
	Log.Infof("Got the pod name=%s running!", name)
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
		Log.Infof("Failed to create secret %s\n", secretName)
		return err
	}
	Log.Infof("Secret %s created\n", secretName)

	// label the secret for use as istio/multiCluster config
	_, err = ShellMuteOutput("kubectl label secret %s %s=%s -n %s --kubeconfig=%s",
		secretName, secretLabel, labelValue, namespace, localKubeConfig)
	if err != nil {
		return err
	}

	Log.Infof("Secret %s labelled with %s=%s\n", secretName, secretLabel, labelValue)
	return nil
}

// DeleteMultiClusterSecret delete the remote cluster secret
func DeleteMultiClusterSecret(namespace string, remoteKubeConfig string, localKubeConfig string) error {
	secretName := filepath.Base(remoteKubeConfig)
	_, err := ShellMuteOutput("kubectl delete secret %s -n %s --kubeconfig=%s", secretName, namespace, localKubeConfig)
	if err != nil {
		Log.Errorf("Failed to delete secret %s: %v", secretName, err)
	} else {
		Log.Infof("Deleted secret %s", secretName)
	}
	return err
}

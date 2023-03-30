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
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

var testRetryTimes = 5

// CreateOCPNamespace create a kubernetes namespace
func CreateOCPNamespace(n string) error {
	if _, err := ShellMuteOutput("oc new-project %s", n); err != nil {
		if !strings.Contains(err.Error(), "AlreadyExists") {
			return err
		}
	}
	log.Log.Infof("namespace %s created\n", n)
	return nil
}

// DeleteOCPNamespace create a kubernetes namespace
func DeleteOCPNamespace(n string) error {
	if _, err := ShellMuteOutput("oc delete project %s", n); err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			return err
		}
	}
	log.Log.Infof("namespace %s deleted\n", n)
	return nil
}

// OcGrantPermission OCP cluster specific requirements for deploying an application with sidecar.
// This is a temporary permission config
func OcGrantPermission(account, namespace string) {
	// Shell("oc adm policy add-scc-to-user privileged -z %s -n %s", account, namespace)
	Shell("oc adm policy add-scc-to-user anyuid -z %s -n %s", account, namespace)
}

// GetOCPIngressgateway returns the OCP cluster ingressgateway host URL.
func GetOCPIngressgateway(podLabel, namespace string) (string, error) {
	ingress, err := Shell("oc get routes -l %s -n %s -o jsonpath='{.items[0].spec.host}'",
		podLabel, namespace)

	for i := 0; i < testRetryTimes; i++ {
		if err == nil {
			break
		}
		time.Sleep(time.Duration(5) * time.Second)
		ingress, err = Shell("oc get routes -l %s -n %s -o jsonpath='{.items[0].spec.host}'",
			podLabel, namespace)
	}
	if err != nil {
		return "", err
	}
	return ingress, nil
}

// GetOCP4Ingressgateway returns OCP4 ingress-ingresssgateway external IP hostname
func GetOCP4Ingressgateway(namespace string) (string, error) {
	ingress, err := Shell("oc -n %s get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'",
		namespace)

	return ingress, err
}

// GetIngressPort returns the http ingressgateway port
func GetIngressPort(namespace, serviceName string) (string, error) {
	port, err := Shell(
		"oc -n %s get service %s -o jsonpath='{.spec.ports[?(@.name==\"http2\")].port}'",
		namespace, serviceName)
	if err != nil {
		return "", err
	}
	port = strings.Trim(port, "'")
	rp := regexp.MustCompile(`^[0-9]{1,5}$`)
	if rp.FindString(port) == "" {
		err = fmt.Errorf("unable to find the http2 port of %s", serviceName)
		log.Log.Warn(err)
		return "", err
	}
	return port, nil
}

// GetSecureIngressPort returns the https ingressgateway port
func GetSecureIngressPort(namespace, serviceName string) (string, error) {
	port, err := Shell(
		"oc -n %s get service %s -o jsonpath='{.spec.ports[?(@.name==\"https\")].port}'",
		namespace, serviceName)
	if err != nil {
		return "", err
	}
	port = strings.Trim(port, "'")
	rp := regexp.MustCompile(`^[0-9]{1,5}$`)
	if rp.FindString(port) == "" {
		err = fmt.Errorf("unable to find the https port of %s", serviceName)
		log.Log.Warn(err)
		return "", err
	}
	return port, nil
}

// GetTCPIngressPort returns the tcp ingressgateway port
func GetTCPIngressPort(namespace, serviceName string) (string, error) {
	port, err := Shell(
		"oc -n %s get service %s -o jsonpath='{.spec.ports[?(@.name==\"tcp\")].port}'",
		namespace, serviceName)
	if err != nil {
		return "", err
	}
	port = strings.Trim(port, "'")
	rp := regexp.MustCompile(`^[0-9]{1,5}$`)
	if rp.FindString(port) == "" {
		err = fmt.Errorf("unable to find the tcp port of %s", serviceName)
		log.Log.Warn(err)
		return "", err
	}
	return port, nil
}

// GetIngressHostIP returns the OCP ingressgateway Host IP address from the OCP router endpoint
func GetIngressHostIP() (string, error) {
	ip, err := Shell("oc get endpoints -n default -l router -o jsonpath='{.items[0].subsets[0].addresses[0].ip}'")
	if err != nil {
		return "", err
	}
	return ip, nil
}

// GetJaegerRoute returns the Jaeger Dashboard route
func GetJaegerRoute(namespace string) (string, error) {
	ingress, err := Shell("oc get routes -n %s -l app=jaeger -o jsonpath='{.items[0].spec.host}'",
		namespace)
	return ingress, err
}

// CheckDeploymentIsReady checks whether the deployment is ready by using `oc wait`
func CheckDeploymentIsReady(namespace, name string, timeout time.Duration) (string, error) {
	return Shell(`oc -n %s wait --for condition=Available deploy/%s --timeout %s`, namespace, name, timeout.String())
}

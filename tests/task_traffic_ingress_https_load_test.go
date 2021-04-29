// Copyright 2020 Red Hat, Inc.
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

package tests

import (
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupIngressLoadTest(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, httpbinTLSGatewayHTTPS, kubeconfig)
	util.ShellMuteOutput("kubectl delete secret %s -n %s", "httpbin-credential", meshNamespace)
	_, err := util.ShellMuteOutput("sudo sed '/httpbin\\.example\\.com/d' -i /etc/hosts")
	if err != nil {
		// error when running inside a container, e.g. sed: cannot rename /etc/seddyBxcL: Device or resource busy
		util.ShellMuteOutput("sudo cp /etc/hosts hosts.new")
		util.ShellMuteOutput("sed '$d' -i hosts.new")
		util.ShellMuteOutput("sudo cp -f hosts.new /etc/hosts")
		util.ShellMuteOutput("rm -f hosts.new")
	}

	cleanHttpbin(namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func TestIngressLoad(t *testing.T) {
	defer cleanupIngressLoadTest(testNamespace)
	defer recoverPanic(t)
	log.Infof("# TestIngressHttpsLoad")
	deployHttpbin(testNamespace)

	if _, err := util.CreateTLSSecret("httpbin-credential", meshNamespace, httpbinSampleServerCertKey, httpbinSampleServerCert, kubeconfig); err != nil {
		t.Errorf("Failed to create secret %s\n", "httpbin-credential")
		log.Infof("Failed to create secret %s\n", "httpbin-credential")
	}

	// config https gateway
	if err := util.KubeApplyContents(testNamespace, httpbinTLSGatewayHTTPS, kubeconfig); err != nil {
		t.Errorf("Failed to configure Gateway")
		log.Errorf("Failed to configure Gateway")
	}
	time.Sleep(time.Duration(waitTime*4) * time.Second)

	t.Run("TrafficManagement_ingress_https_load_test", func(t *testing.T) {
		defer recoverPanic(t)

		// download Jmeter
		util.ShellMuteOutput("curl -Lo ./apache-jmeter-5.3.tgz %s", jmeterURL)
		util.ShellMuteOutput("tar -xzvf apache-jmeter-5.3.tgz")

		// append hosts DNS
		destIP, _ := util.ShellMuteOutput("dig +short $(oc get console.config.openshift.io cluster -o jsonpath='{.status.consoleURL}') | head -n 1 | tr -d '\n'")
		util.ShellMuteOutput(`echo "%s httpbin.example.com" | sudo tee -a /etc/hosts`, destIP)

		// check headers
		url := "https://httpbin.example.com/headers"
		resp, _ := util.ShellMuteOutput("curl -k -v %s", url)
		if strings.Contains(resp, "headers") {
			log.Info(string(resp))
		} else {
			t.Errorf("Failed to get headers: %v", string(resp))
		}

		// Run Load test
		log.Infof("Start load testing. Wait 1 min...")
		util.Shell("apache-jmeter-5.3/bin/jmeter -n -t config/AuthHTTPRequest.jmx.httpbin.xml | tee jmeter-report.out")

		body, _ := ioutil.ReadFile("jmeter-report.out")
		r, _ := regexp.Compile("[1-9]+(\\.[1-9]+)?%")
		if r.MatchString(string(body)) {
			log.Infof("Failed. Err rate is not zero.")
			t.Errorf("Failed. Err rate is not zero.")
		} else {
			log.Infof("Load Test Pass.")
		}
	})
}

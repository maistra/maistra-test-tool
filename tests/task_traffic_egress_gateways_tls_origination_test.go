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
	"strings"
	"testing"
	"time"

	"maistra/util"

	"istio.io/pkg/log"
)

func cleanupEgressGatewaysTLSOrigination(namespace string) {
	log.Info("# Cleanup ...")
	util.KubeDeleteContents(namespace, sleepNginx, kubeconfig)
	util.Shell("kubectl delete -n %s secret nginx-client-certs", namespace)
	util.KubeDeleteContents(namespace, nginxSSLServer, kubeconfig)
	util.Shell("kubectl delete configmap nginx-configmap -n %s", namespace)
	util.Shell("kubectl delete -n %s secret nginx-server-certs", namespace)
	util.Shell("kubectl delete -n %s secret nginx-ca-certs", namespace)
	util.KubeDeleteContents(namespace, cnnextGatewayTLSOrigination, kubeconfig)
	cleanSleep(namespace)
	util.Shell("kubectl get secret -n %s", namespace)
	util.Shell("kubectl get configmap -n %s", namespace)
	time.Sleep(time.Duration(waitTime*4) * time.Second)
}

func TestEgressGatewaysTLSOrigination(t *testing.T) {
	defer cleanupEgressGatewaysTLSOrigination(testNamespace)
	defer recoverPanic(t)

	log.Info("# TestEgressGatewaysTLSOrigination")
	deploySleep(testNamespace)
	sleepPod, err := util.GetPodName(testNamespace, "app=sleep", kubeconfig)
	util.Inspect(err, "Failed to get sleep pod name", "", t)

	t.Run("TrafficManagement_egress_gateway_perform_TLS_origination", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("Perform TLS origination with an egress gateway")
		util.KubeApplyContents(testNamespace, cnnextGatewayTLSOrigination, kubeconfig)
		// OCP Route created by ior
		time.Sleep(time.Duration(waitTime*4) * time.Second)
		command := "curl -sL -o /dev/null -D - http://edition.cnn.com/politics"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "HTTP/1.1 200 OK") {
			log.Infof("Success. Get http://edition.cnn.com/politics response: %s", msg)

		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}

		util.KubeDeleteContents(testNamespace, cnnextGatewayTLSOrigination, kubeconfig)
		cleanSleep(testNamespace)
		time.Sleep(time.Duration(waitTime*4) * time.Second)
	})

	t.Run("TrafficManagement_egress_gateway_perform_MTLS_origination", func(t *testing.T) {
		defer recoverPanic(t)

		log.Info("deploy nginx sever")
		util.Shell("kubectl create -n %s secret tls nginx-server-certs --key %s --cert %s", testNamespace, nginxServerCertKey, nginxServerCert)
		util.Shell("kubectl create -n %s secret generic nginx-ca-certs --from-file=%s", testNamespace, nginxServerCACert)
		util.Shell("kubectl create configmap nginx-configmap -n %s --from-file=nginx.conf=%s", testNamespace, "config/nginx_ssl.conf")
		time.Sleep(time.Duration(waitTime) * time.Second)
		util.KubeApplyContents(testNamespace, nginxSSLServer, kubeconfig)
		util.CheckPodRunning(testNamespace, "run=my-nginx", kubeconfig)

		log.Info("deploy a client")
		util.Shell("kubectl create -n %s secret tls nginx-client-certs --key %s --cert %s", testNamespace, nginxClientCertKey, nginxClientCert)
		//util.Shell("kubectl create -n %s secret generic nginx-ca-certs --from-file=%s", testNamespace, nginxServerCACert)
		util.KubeApplyContents(testNamespace, sleepNginx, kubeconfig)
		util.CheckPodRunning(testNamespace, "app=sleep", kubeconfig)
		time.Sleep(time.Duration(waitTime*2) * time.Second)

		sleepPod, err = util.GetPodName(testNamespace, "app=sleep", kubeconfig)
		util.Inspect(err, "Failed to get sleep pod name", "", t)

		command := "curl -v --resolve nginx.example.com:443:1.1.1.1 --cacert /etc/nginx-ca-certs/example.com.crt --cert /etc/nginx-client-certs/tls.crt --key /etc/nginx-client-certs/tls.key https://nginx.example.com"
		msg, err := util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "200 OK") {
			log.Infof("Success. Get https://nginx.example.com response: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
		command = "curl -k --resolve nginx.example.com:443:1.1.1.1 https://nginx.example.com"
		msg, err = util.PodExec(testNamespace, sleepPod, "sleep", command, false, kubeconfig)
		util.Inspect(err, "Failed to get response", "", t)
		if strings.Contains(msg, "400") {
			log.Infof("Success. Get expected 400 Bad Request: %s", msg)
		} else {
			log.Infof("Error response: %s", msg)
			t.Errorf("Error response: %s", msg)
		}
	})
}

// Copyright 2019 Red Hat, Inc.
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

package maistra

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup21(namespace string, kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.Shell("oc delete secret cacerts -n " + meshNamespace)
	util.Shell("oc rollout undo deployment istio-citadel -n " + meshNamespace)
	util.ShellMuteOutput("rm -f /tmp/istio-citadel-new.yaml")
	cleanBookinfo(namespace, kubeconfig)
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}


func verifyCerts() error {
	pod, err := util.GetPodName(testNamespace, "app=ratings", kubeconfigFile)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("/bin/cat /etc/certs/root-cert.pem")
	podRootCert, err := util.PodExec(testNamespace, pod, "istio-proxy", cmd, true, kubeconfigFile)
	if err != nil {
		return err
	}
	ioutil.WriteFile("/tmp/pod-root-cert.pem", []byte(podRootCert), 0644)
	//util.ShellMuteOutput("oc exec -n %s -it %s -c istio-proxy -- /bin/cat /etc/certs/root-cert.pem > /tmp/pod-root-cert.pem", testNamespace, pod)
	
	cmd = fmt.Sprintf("/bin/cat /etc/certs/cert-chain.pem")
	podCertChain, err := util.PodExec(testNamespace, pod, "istio-proxy", cmd, true, kubeconfigFile)
	if err != nil {
		return err
	}
	ioutil.WriteFile("/tmp/pod-cert-chain.pem", []byte(podCertChain), 0644)
	//util.ShellMuteOutput("oc exec -n %s -it %s -c istio-proxy -- /bin/cat /etc/certs/cert-chain.pem > /tmp/pod-cert-chain.pem", testNamespace, pod)

	util.ShellMuteOutput("openssl x509 -in %s -text -noout > /tmp/root-cert.crt.txt", caRootCert)
	util.ShellMuteOutput("openssl x509 -in %s -text -noout > /tmp/pod-root-cert.crt.txt", "/tmp/pod-root-cert.pem")
	err = util.CompareFiles("/tmp/root-cert.crt.txt", "/tmp/pod-root-cert.crt.txt")
	if err != nil {
		return err
	}

	util.ShellMuteOutput("tail -n 22 %s > /tmp/pod-cert-chain-ca.pem", "/tmp/pod-cert-chain.pem")
	util.ShellMuteOutput("openssl x509 -in %s -text -noout > /tmp/ca-cert.crt.txt", caCert)
	util.ShellMuteOutput("openssl x509 -in /tmp/pod-cert-chain-ca.pem -text -noout > /tmp/pod-cert-chain-ca.crt.txt")
	err = util.CompareFiles("/tmp/ca-cert.crt.txt", "/tmp/pod-cert-chain-ca.crt.txt")
	if err != nil {
		return err
	}

	util.ShellMuteOutput("head -n 21 %s > /tmp/pod-cert-chain-workload.pem", "/tmp/pod-cert-chain.pem")
	util.ShellMuteOutput("cat %s %s > /tmp/ca-cert-file.crt.txt", caCert, caRootCert)
	msg, err := util.Shell("openssl verify -CAfile /tmp/ca-cert-file.crt.txt /tmp/pod-cert-chain-workload.pem")
	if err != nil || !strings.Contains(msg, "OK") {
		return fmt.Errorf("Error certs: %s", msg)
	}

	return nil
}


func Test21mtls(t *testing.T) {

	defer cleanup21(testNamespace, kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	t.Run("plugging_external_certs_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Infof("# TC_21 Plugging in External CA Key and Certificate")

		util.CreateNamespace(testNamespace, kubeconfigFile)
		//util.OcGrantPermission("default", testNamespace, kubeconfigFile)

		log.Info("Create secret")
		_, err := util.ShellMuteOutput("oc create secret generic %s -n %s --from-file %s --from-file %s --from-file %s --from-file %s --kubeconfig=%s", 
								"cacerts",
								meshNamespace,
								caCert,
								caCertKey,
								caRootCert,
								caCertChain,
								kubeconfigFile)
		if err != nil {
			log.Infof("Failed to create secret %s\n", "cacerts")
			t.Errorf("Failed to create secret %s\n", "cacerts")
		}
		log.Infof("Secret %s created\n", "cacerts")
		time.Sleep(time.Duration(5) * time.Second)

		log.Info("Redeploy Citadel")
		backupFile := "/tmp/istio-citadel-bak.yaml"
		newFile := "/tmp/istio-citadel-new.yaml"

		util.ShellMuteOutput("oc get deployment -n %s %s -o yaml --kubeconfig=%s > %s",
						meshNamespace,
						"istio-citadel",
						kubeconfigFile,
						backupFile)
	
		data, err := ioutil.ReadFile(backupFile)
		if err != nil {
			log.Infof("Unable to read citadel deployment yaml: %v", err)
			t.Errorf("Unable to read citadel deployment yaml: %v", err)
		}
		w, _ := os.Create(newFile)
		defer w.Close()
		err = util.ConfigCitadelCerts(data, w)
		if err != nil {
			log.Infof("Update citadel deployment error: %v", err)
			t.Errorf("Update citadel deployment error: %v", err)
		}
		util.Shell("oc apply -n %s -f %s", meshNamespace, newFile)
		time.Sleep(time.Duration(20) * time.Second)
		
		log.Info("Delete existing istio.default secret")
		util.Shell("oc delete -n %s secret istio.default", testNamespace)

		log.Info("Deploy bookinfo")
		util.Inspect(deployBookinfo(testNamespace, kubeconfigFile, true), "failed to deploy bookinfo", "Bookinfo deployment completed", t)
		time.Sleep(time.Duration(20) * time.Second)
		
		log.Info("Verify certs")
		err = verifyCerts()
		if err != nil {
			log.Infof("%v", err)
			t.Errorf("%v", err)
		}
	})

} 

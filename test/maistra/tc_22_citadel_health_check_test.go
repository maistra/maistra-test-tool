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
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"istio.io/istio/pkg/log"
	"maistra/util"
)

func cleanup22(kubeconfig string) {
	log.Infof("# Cleanup. Following error can be ignored...")
	util.Shell("oc rollout undo deployment istio-citadel -n " + meshNamespace)
	util.ShellMuteOutput("rm -f /tmp/istio-citadel-new.yaml")
	log.Info("Waiting... Sleep 20 seconds...")
	time.Sleep(time.Duration(20) * time.Second)
}


func Test22mtls(t *testing.T) {
	defer cleanup22(kubeconfigFile)
	defer func() {
		// recover from panic if one occured. This allows cleanup to be executed after panic.
		if err := recover(); err != nil {
			t.Errorf("Test panic: %v", err)
		}
	}()

	t.Run("citadel_health_check_test", func(t *testing.T) {
		defer func() {
			// recover from panic if one occured. This allows cleanup to be executed after panic.
			if err := recover(); err != nil {
				t.Errorf("Test panic: %v", err)
			}
		}()

		log.Info("# TC_22 Citadel Health Checking")
		
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
		newdata := strings.Replace(string(data), "extensions/v1beta1", "apps/v1", -1)

		w, _ := os.Create(newFile)
		defer w.Close()
		
		err = util.ConfigCitadelHealthCheck([]byte(newdata), w)
		if err != nil {
			log.Infof("Update citadel deployment error: %v", err)
			t.Errorf("Update citadel deployment error: %v", err)
		}
		util.Shell("oc apply -n %s -f %s", meshNamespace, newFile)
		log.Info("Waiting... Sleep 30 seconds...")
		time.Sleep(time.Duration(30) * time.Second)

		pod, err := util.GetPodName(meshNamespace, "istio=citadel", kubeconfigFile)
		util.Inspect(err, "failed to get sleep pod name", "", t)
		msg, _ := util.ShellMuteOutput("oc logs -n %s %s", meshNamespace, pod)
		if !strings.Contains(msg, "CSR") {
			log.Infof("Error no CSR is healthy log")
			t.Errorf("Error no CSR is healthy log")
		} else {
			re, _ := regexp.Compile(".*CSR.*")
			log.Infof("%v", re.FindString(msg))
		}
	})

}

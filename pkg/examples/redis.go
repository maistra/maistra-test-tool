// Copyright Red Hat, Inc.
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
	"fmt"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

type Redis struct {
	Namespace string `json:"namespace,omitempty"`
}

func (r *Redis) Install() error {
	util.Log.Info("Deploy Redis")

	if err := util.CreateOCPNamespace(r.Namespace); err != nil {
		return fmt.Errorf("error creating redis namespace: %v", err)
	}

	if err := util.KubeApply(r.Namespace, redisYaml); err != nil {
		return fmt.Errorf("error deploying redis: %v", err)
	}

	if _, err := util.Shell(`oc -n %s wait --for condition=Available deploy/redis --timeout 180s`, r.Namespace); err != nil {
		return fmt.Errorf("redis deployment not ready: %v", err)
	}

	if err := util.CheckPodRunning(r.Namespace, "app=redis"); err != nil {
		return fmt.Errorf("redis deployment not ready: %v", err)
	}

	return nil
}

func (r *Redis) Uninstall() {
	util.Log.Info("Cleanup Redis")
	util.KubeDelete(r.Namespace, redisYaml)
	util.DeleteNamespace(r.Namespace)
	time.Sleep(time.Duration(10) * time.Second)
}

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
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

// samples directory is github.com/maistra/maistra-test-tool/testdata/examples/x86

// Bookinfo includes app deployment namespace
type Bookinfo struct {
	Namespace string `json:"namespace,omitempty"`
}

func (b *Bookinfo) Install(mtls bool) {
	log.Log.Info("Deploying Bookinfo")
	util.KubeApply(b.Namespace, bookinfoYaml)
	time.Sleep(time.Duration(5) * time.Second)
	util.CheckPodRunning(b.Namespace, "app=details")
	util.CheckPodRunning(b.Namespace, "app=ratings")
	util.CheckPodRunning(b.Namespace, "app=reviews,version=v1")
	util.CheckPodRunning(b.Namespace, "app=reviews,version=v2")
	util.CheckPodRunning(b.Namespace, "app=reviews,version=v3")
	util.CheckPodRunning(b.Namespace, "app=productpage")

	log.Log.Info("Creating Gateway")
	util.KubeApply(b.Namespace, bookinfoGateway)

	log.Log.Info("Creating destination rules all")
	if mtls {
		util.KubeApply(b.Namespace, bookinfoRuleAllTLSYaml)
	} else {
		util.KubeApply(b.Namespace, bookinfoRuleAllYaml)
	}
	time.Sleep(time.Duration(10) * time.Second)
}

func (b *Bookinfo) Uninstall() {
	log.Log.Info("Cleanup Bookinfo")
	util.KubeDelete(b.Namespace, bookinfoRuleAllYaml)
	util.KubeDelete(b.Namespace, bookinfoGateway)
	util.KubeDelete(b.Namespace, bookinfoYaml)
	time.Sleep(time.Duration(10) * time.Second)
}

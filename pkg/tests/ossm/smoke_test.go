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

package ossm

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/app"
	. "github.com/maistra/maistra-test-tool/pkg/examples"
	. "github.com/maistra/maistra-test-tool/pkg/util"
	. "github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func cleanupBookinfo() {
	Log.Info("Cleanup")
	app := Bookinfo{"bookinfo"}
	app.Uninstall()
	time.Sleep(time.Duration(30) * time.Second)
}

func TestBookinfo(t *testing.T) {
	NewTest(t).Id("A2").Groups(ARM, Full, Smoke, InterOp).Run(func(t TestHelper) {
		Log.Info("Test Bookinfo Installation")
		app.InstallAndWaitReady(t, app.Bookinfo("bookinfo"))

		Log.Info("Check pods running 2/2 ready")
		msg, _ := Shell(`oc get pods -n bookinfo`)
		if strings.Contains(msg, "2/2") {
			Log.Info("Success. proxy container is running.")
		} else {
			t.Error("Error. proxy container is not running.")
		}

		Log.Info("Check istiod pod is ready and print istiod logs")
		mesg, _ := Shell(`oc get pods -n istio-system | grep istiod`)
		if strings.Contains(mesg, "1/1") {
			Log.Info("Success. istiod pod is running with below logs:")
			shell.Executef(t, `oc logs -n %s -l app=istiod | grep info`, meshNamespace)
		} else {
			t.Error("Error. istiod pod is not running.")
		}

		Log.Info("Check if bookinfo productpage is running")
		GATEWAY_URL, _ := Shell(`oc -n %s get route istio-ingressgateway -o jsonpath='{.spec.host}'`, meshNamespace)
		mes, _ := Shell(`curl -o /dev/null -s -w "%%{http_code}\n" http://%s/productpage`, GATEWAY_URL)
		if strings.Contains(mes, "200") {
			Log.Info("Success. bookinfo productpage is running")
		} else {
			t.Error("Error. bookinfo productpage is not running.")
			Log.Error("Error. bookinfo productpage is not running.")
		}
	})
}

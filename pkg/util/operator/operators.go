// Copyright 2024 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"fmt"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func GetFullCsvName(t test.TestHelper, namespace string, partialCsvName string) string {
	output := shell.Execute(t, fmt.Sprintf(`oc get csv -n %s -o custom-columns="NAME:.metadata.name" |grep %s ||true`, namespace, partialCsvName))
	return strings.TrimSpace(output)
}

func WaitForOperatorInNamespaceReady(t test.TestHelper, namespace string, operatorSelector string, partialCsvName string) {
	t.Logf("Waiting for operator csv %s to succeed", partialCsvName)
	// When the operator is installed, the CSV take some time to be created, need to wait until is created to validate the phase
	retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(5*time.Second).MaxAttempts(70), func(t test.TestHelper) {
		if !operatorCsvExistsGlobally(t, partialCsvName) {
			t.Errorf("Operator csv %s is not yet installed", partialCsvName)
		}
	})

	csvFullName := GetFullCsvName(t, namespace, partialCsvName)
	oc.WaitForPhase(t, namespace, "csv", csvFullName, "Succeeded")
	t.Logf("Waiting for operator pod with the selector %s", operatorSelector)
	oc.WaitPodReadyWithOptions(t, retry.Options().MaxAttempts(70).DelayBetweenAttempts(5*time.Second), pod.MatchingSelector(operatorSelector, namespace))
}

func CreateOperatorViaOlm(t test.TestHelper, namespace string, partialCsvName string, subscriptionYaml string, operatorPodSelector string, input interface{}) {
	oc.ApplyTemplate(t, namespace, subscriptionYaml, input)
	WaitForOperatorInNamespaceReady(t, namespace, operatorPodSelector, partialCsvName)
}

func DeleteOperatorViaOlm(t test.TestHelper, namespace string, partialCsvName string, subscriptionYaml string) {
	csvFullName := GetFullCsvName(t, namespace, partialCsvName)
	t.Logf("Deleting subscription for %s", csvFullName)
	oc.DeleteFromTemplate(t, namespace, subscriptionYaml, nil)
	t.Logf("Deleting csv %s", csvFullName)
	oc.DeleteResource(t, namespace, "csv", csvFullName)
}

// func waitForCsvReady(t test.TestHelper, partialName string) {
// 	t.Logf("Waiting for csv %s is ready", partialName)
// 	retry.UntilSuccessWithOptions(t, retry.Options().DelayBetweenAttempts(1*time.Second).MaxAttempts(20), func(t test.TestHelper) {
// 		output := shell.Execute(t, fmt.Sprintf(`oc get csv -A -o custom-columns="NAME:.metadata.name" |grep %s ||true`, partialName))
// 		if output == "" {
// 			t.Errorf("CSV %s is not ready yet", partialName)
// 		}
// 	})
// }

func operatorCsvExistsGlobally(t test.TestHelper, csvVersion string) bool {
	output := shell.Execute(t, fmt.Sprintf(`oc get csv -A -o custom-columns="NAME:.metadata.name" |grep %s ||true`, csvVersion))
	return strings.Contains(output, csvVersion)
}

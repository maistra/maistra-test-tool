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

package prometheusoperator

import (
	_ "embed"
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/operator"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/custom-prometheus-operator.yaml
	prometheusSubscriptionYaml string

	//go:embed yaml/prometheus-instance.yaml
	prometheusInstanceYaml string

	customPrometheusNamespace = "custom-prometheus-operator"

	prometheusCsvName          = "prometheusoperator"
	prometheusOperatorSelector = "app.kubernetes.io/name=prometheus-operator"
)

func Install(t test.TestHelper) {
	oc.CreateNamespace(t, customPrometheusNamespace)
	t.Log("Instaling custom prometheus operator...")
	operator.CreateOperatorViaOlm(t, customPrometheusNamespace, prometheusCsvName, prometheusSubscriptionYaml, prometheusOperatorSelector, nil)
}

func Uninstall(t test.TestHelper) {
	t.Log("Uninstalling custom prometheus")
	oc.DeleteFromTemplate(t, customPrometheusNamespace, prometheusInstanceYaml, nil)
	operator.DeleteOperatorViaOlm(t, customPrometheusNamespace, prometheusCsvName, prometheusSubscriptionYaml)
	oc.DeleteNamespace(t, customPrometheusNamespace)
}

func InstalPrometheusInstance(t test.TestHelper, permittedNs ...string) {
	oc.ApplyTemplate(t, customPrometheusNamespace, prometheusInstanceYaml, nil)
	t.Log("Waiting for custom prometheus to be ready")
	oc.DefaultOC.WaitFor(t, customPrometheusNamespace, "Prometheus", "prometheus", "condition=Reconciled")

	for _, permitNs := range permittedNs {
		oc.ApplyString(t, permitNs,
			`
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: custom-prometheus-permissions
rules:
- apiGroups: [""]
  resources:
  - services
  - endpoints
  - pods
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources:
  - configmaps
  verbs: ["get"]`,
			fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: custom-prometheus-permissions
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: custom-prometheus-permissions
subjects:
- kind: ServiceAccount
  name: prometheus-k8s
  namespace: %s`, customPrometheusNamespace))
	}
	retry.UntilSuccess(t, func(t test.TestHelper) {
		prometheusPod := pod.MatchingSelector("app.kubernetes.io/name=prometheus-operator", customPrometheusNamespace)
		oc.WaitPodRunning(t, prometheusPod)
	})
}

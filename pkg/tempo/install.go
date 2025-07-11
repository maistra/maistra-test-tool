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

package tempo

import (
	_ "embed"

	"github.com/maistra/maistra-test-tool/pkg/app"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/operator"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

var (
	//go:embed yaml/tempo-operator.yaml
	tempoSubscriptionYaml string

	//go:embed yaml/open-telemetry-operator.yaml
	optelSubscriptionYaml string

	//go:embed yaml/tempoStack.yaml
	tempoStack string

	tempoOperatorsNamespace = "openshift-tempo-operator"
	otelOperatorsNamespace  = "openshift-opentelemetry-operator"

	// sometimes tempo/otel operators can be in the openshift-operators namespace
	openshiftOperatorsNamespace = "openshift-operators"

	tracingNamespace = "tracing-system"

	tempoCsvName          = "tempo-operator"
	tempoOperatorSelector = "app.kubernetes.io/name=tempo-operator"
	otelCsvName           = "opentelemetry-operator"
	otelOperatorSelector  = "app.kubernetes.io/name=opentelemetry-operator"
)

var tempoExistedBefore = false
var otelExistedBefore = false

func InstallIfNotExist(t test.TestHelper) {
	if oc.ResourceByLabelExists(t, tempoOperatorsNamespace, "pod", tempoOperatorSelector) || oc.ResourceByLabelExists(t, openshiftOperatorsNamespace, "pod", tempoOperatorSelector) {
		t.Log("Tempo operator is already installed")
		tempoExistedBefore = true
	} else {
		oc.CreateNamespace(t, tempoOperatorsNamespace)
		t.Log("Tempo operator is not installed, installing...")
		operator.CreateOperatorViaOlm(t, tempoOperatorsNamespace, tempoCsvName, tempoSubscriptionYaml, tempoOperatorSelector, nil)
	}

	if oc.ResourceByLabelExists(t, otelOperatorsNamespace, "pod", otelOperatorSelector) || oc.ResourceByLabelExists(t, openshiftOperatorsNamespace, "pod", otelOperatorSelector) {
		t.Log("OpenTelemetry operator is already installed")
		otelExistedBefore = true
	} else {
		t.Log("OpenTelemetry operator is not installed, installing...")
		oc.CreateNamespace(t, otelOperatorsNamespace)
		operator.CreateOperatorViaOlm(t, otelOperatorsNamespace, otelCsvName, optelSubscriptionYaml, otelOperatorSelector, nil)
	}

	if oc.ResourceByLabelExists(t, tracingNamespace, "pod", "app=minio") && oc.ResourceByLabelExists(t, tracingNamespace, "pod", "app.kubernetes.io/component=query-frontend") {
		t.Log("TempoStack CR is already installed")
	} else {
		t.Log("TempoStack CR is not installed or it is corrupted, installing...")
		installTempoStack(t)
	}
}

func Uninstall(t test.TestHelper) {
	t.Log("Uninstalling TempoStack")
	oc.DeleteResource(t, tracingNamespace, "Route", "tracing-ui")
	oc.DeleteFromTemplate(t, tracingNamespace, tempoStack, nil)
	app.Uninstall(t, app.Minio(tracingNamespace))
	oc.DeleteNamespace(t, tracingNamespace)
	if !tempoExistedBefore {
		t.Log("Uninstalling Tempo operator")
		operator.DeleteOperatorViaOlm(t, tempoOperatorsNamespace, tempoCsvName, tempoSubscriptionYaml)
		oc.DeleteNamespace(t, tempoOperatorsNamespace)
	} else {
		t.Log("Tempo operator was existed before testing, uninstalling skipped!")
	}

	if !otelExistedBefore {
		t.Log("Uninstalling Otel operator")
		operator.DeleteOperatorViaOlm(t, otelOperatorsNamespace, otelCsvName, optelSubscriptionYaml)
		oc.DeleteNamespace(t, otelOperatorsNamespace)
	} else {
		t.Log("Otel operator was existed before testing, uninstalling skipped!")
	}
}

func GetTracingNamespace() string {
	return tracingNamespace
}

func GetFrontEndQueryRouteUrl(t test.TestHelper) string {
	return oc.DefaultOC.GetRouteURL(t, tracingNamespace, "tracing-ui")
}

func installTempoStack(t test.TestHelper) {
	oc.RecreateNamespace(t, tracingNamespace)
	app.InstallAndWaitReady(t, app.Minio(tracingNamespace))
	oc.ApplyTemplate(t, tracingNamespace, tempoStack, nil)
	t.Log("Waiting for TempoStack to be ready")
	oc.DefaultOC.WaitFor(t, tracingNamespace, "TempoStack", "sample", "condition=Ready")
	t.Log("Waiting for TempoStack to be ready")
	oc.WaitDeploymentRolloutComplete(t, tracingNamespace, "tempo-sample-compactor")
	// just to be sure that no hanging tracing ui route exists
	oc.DeleteResource(t, tracingNamespace, "Route", "tracing-ui")
	oc.ExposeSvc(t, tracingNamespace, "tempo-sample-query-frontend", "jaeger-ui", "tracing-ui")
}

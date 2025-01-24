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

package helm

import (
	"fmt"

	"github.com/maistra/maistra-test-tool/pkg/util/shell"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type cmd struct {
	chart       string
	namespace   string
	release     string
	setValues   []string
	valuesStdIn string
	version     string
}

func Namespace(ns string) cmd {
	return cmd{namespace: ns}
}

func (c cmd) Release(name string) cmd {
	c.release = name
	return c
}

func (c cmd) Chart(chart string) cmd {
	c.chart = chart
	return c
}

func (c cmd) Version(ver string) cmd {
	c.version = ver
	return c
}

func (c cmd) Set(keyValue string) cmd {
	c.setValues = append(c.setValues, keyValue)
	return c
}

func (c cmd) ValuesString(values string) cmd {
	c.valuesStdIn = values
	return c
}

func (c cmd) Install(t test.TestHelper) {
	if c.chart == "" {
		t.Fatalf("chart must be specified to install helm release")
	}
	if c.release == "" {
		t.Fatalf("release name must be specified to install helm release")
	}
	if c.namespace == "" {
		t.Fatalf("namespace must be specified to install helm release")
	}
	cmd := fmt.Sprintf("helm install %s %s -n %s", c.release, c.chart, c.namespace)
	if c.version != "" {
		cmd += fmt.Sprintf(" --version %s", c.version)
	}
	for _, val := range c.setValues {
		cmd += fmt.Sprintf(" --set %s", val)
	}
	if c.valuesStdIn != "" {
		cmd += fmt.Sprintf(` -f - <<EOF
%s
EOF`, c.valuesStdIn)
	}
	shell.Execute(t, cmd)
}

func (c cmd) Uninstall(t test.TestHelper) {
	if c.release == "" {
		t.Fatalf("release name must be specified to uninstall helm release")
	}
	if c.namespace == "" {
		t.Fatalf("namespace must be specified to uninstall helm release")
	}
	shell.Executef(t, "helm uninstall %s -n %s --ignore-not-found", c.release, c.namespace)
}

type repo struct {
	url string
}

func Repo(url string) repo {
	return repo{url: url}
}

func (r repo) Add(t test.TestHelper, name string) {
	if name == "" {
		t.Fatalf("repo name must be specified")
	}
	shell.Executef(t, "helm repo add %s %s", name, r.url)
}

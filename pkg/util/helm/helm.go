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
	shell.Executef(t, "helm uninstall %s -n %s", c.release, c.namespace)
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

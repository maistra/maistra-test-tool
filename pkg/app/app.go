package app

import (
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type App interface {
	Name() string
	Namespace() string
	Install(t test.TestHelper)
	Uninstall(t test.TestHelper)
	WaitReady(t test.TestHelper)
}

func InstallAndWaitReady(t test.TestHelper, apps ...App) {
	t.T().Helper()
	Install(t, apps...)
	WaitReady(t, apps...)
}

func Install(t test.TestHelper, apps ...App) {
	t.T().Helper()
	for _, app := range apps {
		t.Logf("Installing app %q in namespace %q", app.Name(), app.Namespace())
		app.Install(t)
	}
}

func WaitReady(t test.TestHelper, apps ...App) {
	t.T().Helper()
	for _, app := range apps {
		t.Logf("Waiting for app %q in namespace %q to be ready", app.Name(), app.Namespace())
		app.WaitReady(t)
	}
}

func Uninstall(t test.TestHelper, apps ...App) {
	t.T().Helper()
	for _, app := range apps {
		app.Uninstall(t)
	}
}

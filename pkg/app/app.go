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
		t.Logf("Install app %q in namespace %q", app.Name(), app.Namespace())
		app.Install(t)
	}
}

func WaitReady(t test.TestHelper, apps ...App) {
	t.T().Helper()
	for _, app := range apps {
		t.Logf("Wait for app %s/%s to be ready", app.Namespace(), app.Name())
		app.WaitReady(t)
	}
}

func Uninstall(t test.TestHelper, apps ...App) {
	t.T().Helper()
	for _, app := range apps {
		app.Uninstall(t)
	}
}

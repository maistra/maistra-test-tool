// Copyright 2023 Red Hat, Inc.
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

package app

import (
	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type echoV1 struct {
	ns string
}

var _ App = &echoV1{}

func EchoV1(ns string) App {
	return &echoV1{ns: ns}
}

func (a *echoV1) Name() string {
	return "echo-v1"
}

func (a *echoV1) Namespace() string {
	return a.ns
}

func (a *echoV1) Install(t test.TestHelper) {
	t.T().Helper()
	oc.ApplyFile(t, a.ns, examples.EchoV1YamlFile())
}

func (a *echoV1) Uninstall(t test.TestHelper) {
	t.T().Helper()
	oc.DeleteFile(t, a.ns, examples.EchoV1YamlFile())
}

func (a *echoV1) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "tcp-echo-v1")
}

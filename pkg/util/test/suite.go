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

package test

import (
	"os"
	"testing"
)

type SetupFunc func(t TestHelper)

type TestSuite interface {
	Run()
	Setup(SetupFunc) TestSuite
}

type testSuite struct {
	m        *testing.M
	setupFns []SetupFunc
}

func (s *testSuite) Run() {
	t := NewSetupTestHelper().(*setupTestHelper)
	for _, setupFn := range s.setupFns {
		setupFn(t)
	}

	exitCode := s.m.Run()
	if t.cleanup != nil {
		t.cleanup()
	}
	os.Exit(exitCode)
}

func (s *testSuite) Setup(f SetupFunc) TestSuite {
	s.setupFns = append(s.setupFns, f)
	return s
}

func NewSuite(m *testing.M) TestSuite {
	return &testSuite{m: m}
}

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
	t := NewSetupTestHelper()
	for _, setupFn := range s.setupFns {
		setupFn(t)
	}

	exitCode := s.m.Run()
	t.(*setupTestHelper).cleanup()
	os.Exit(exitCode)
}

func (s *testSuite) Setup(f SetupFunc) TestSuite {
	s.setupFns = append(s.setupFns, f)
	return s
}

func NewSuite(m *testing.M) TestSuite {
	return &testSuite{m: m}
}

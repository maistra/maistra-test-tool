package test

import (
	"testing"
)

func TestSuccessful(t *testing.T) {
	NewTestWithFlakinessDetection(t).Run(func(t TestHelper) {
		t.Log("I always succeed")
		t.Log("success")
	})
}

func TestFailing(t *testing.T) {
	NewTestWithFlakinessDetection(t).Run(func(t TestHelper) {
		t.Log("I always fail")
		t.Error("error")
	})
}

func TestFlaky(t *testing.T) {
	i := 0
	NewTestWithFlakinessDetection(t).Run(func(t TestHelper) {
		t.Log("I sometimes fail")
		i++
		if i%3 == 0 {
			t.Log("success")
		} else {
			t.Error("error")
		}
	})
}

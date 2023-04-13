package version

import "testing"

func TestLessThan(t *testing.T) {
	assertTrue(t, SMCP_2_1.LessThan(SMCP_2_2))
	assertTrue(t, SMCP_2_1.LessThan(SMCP_2_3))
	assertFalse(t, SMCP_2_1.LessThan(SMCP_2_1))
	assertFalse(t, SMCP_2_2.LessThan(SMCP_2_1))
}

func TestGreaterThanOrEqual(t *testing.T) {
	assertFalse(t, SMCP_2_1.GreaterThanOrEqualTo(SMCP_2_2))
	assertFalse(t, SMCP_2_1.GreaterThanOrEqualTo(SMCP_2_3))
	assertTrue(t, SMCP_2_1.GreaterThanOrEqualTo(SMCP_2_1))
	assertTrue(t, SMCP_2_2.GreaterThanOrEqualTo(SMCP_2_1))
}

func assertTrue(t *testing.T, b bool) {
	t.Helper()
	if !b {
		t.Errorf("expected true, but was false")
	}
}

func assertFalse(t *testing.T, b bool) {
	t.Helper()
	if b {
		t.Errorf("expected false, but was true")
	}
}

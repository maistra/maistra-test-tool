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

package version

import "testing"

func TestParseVersion(t *testing.T) {
	assertVersionParsedTo(t, "v2.3", SMCP_2_3)
	assertVersionParsedTo(t, "2.3", SMCP_2_3)
}

func assertVersionParsedTo(t *testing.T, str string, expectedVersion Version) {
	v := ParseVersion(str)
	if v != expectedVersion {
		t.Fatalf("expected %q to be parsed to %q, but was: %q", str, expectedVersion, v)
	}
}
func TestEquals(t *testing.T) {
	assertTrue(t, SMCP_2_1.Equals(SMCP_2_1))
	assertFalse(t, SMCP_2_1.Equals(SMCP_2_2))
	assertFalse(t, SMCP_2_2.Equals(SMCP_2_1))
}

func TestLessThan(t *testing.T) {
	assertTrue(t, SMCP_2_1.LessThan(SMCP_2_2))
	assertTrue(t, SMCP_2_1.LessThan(SMCP_2_3))
	assertFalse(t, SMCP_2_1.LessThan(SMCP_2_1))
	assertFalse(t, SMCP_2_2.LessThan(SMCP_2_1))
}

func TestLessThanOrEqual(t *testing.T) {
	assertTrue(t, SMCP_2_1.LessThanOrEqual(SMCP_2_2))
	assertTrue(t, SMCP_2_1.LessThanOrEqual(SMCP_2_3))
	assertTrue(t, SMCP_2_1.LessThanOrEqual(SMCP_2_1))
	assertFalse(t, SMCP_2_2.LessThanOrEqual(SMCP_2_1))
}

func TestGreaterThan(t *testing.T) {
	assertFalse(t, SMCP_2_1.GreaterThan(SMCP_2_2))
	assertFalse(t, SMCP_2_1.GreaterThan(SMCP_2_3))
	assertFalse(t, SMCP_2_1.GreaterThan(SMCP_2_1))
	assertTrue(t, SMCP_2_2.GreaterThan(SMCP_2_1))
}

func TestGreaterThanOrEqual(t *testing.T) {
	assertFalse(t, SMCP_2_1.GreaterThanOrEqual(SMCP_2_2))
	assertFalse(t, SMCP_2_1.GreaterThanOrEqual(SMCP_2_3))
	assertTrue(t, SMCP_2_1.GreaterThanOrEqual(SMCP_2_1))
	assertTrue(t, SMCP_2_2.GreaterThanOrEqual(SMCP_2_1))
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

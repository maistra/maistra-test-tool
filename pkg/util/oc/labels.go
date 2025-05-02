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

package oc

const (
	// MaistraTestLabel is the label added to resources that are created by the maistra test tool.
	// These resources are labeled so they can easily be purged at the beginning or end of test runs.
	MaistraTestLabel = "maistra.io/maistra-test-tool"
	// A test bound namespace is one that is expected to be deleted after each test run.
	// Probably all namespaces fall into this category.
	testBoundNSLabelValue = "test-bound-ns"
	// testBoundNamespacesSelector match namespaces created by Maistra Test Tool that should be
	// deleted before/after each test.
	testBoundNamespacesSelector = MaistraTestLabel + "=" + testBoundNSLabelValue
)

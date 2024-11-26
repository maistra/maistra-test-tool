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

package pod

import (
	ocpackage "github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func getPods(t test.TestHelper, oc *ocpackage.OC, selector string, ns string) []ocpackage.NamespacedName {
	t.T().Helper()
	pods := oc.GetAllResourcesNames(t, ns, "pods", selector)
	var namespacedNames []ocpackage.NamespacedName
	for _, pod := range pods {
		namespacedNames = append(namespacedNames, ocpackage.NewNamespacedName(ns, pod))
	}
	return namespacedNames
}

func MatchingSelector(selector string, ns string) ocpackage.PodLocatorFunc {
	return func(t test.TestHelper, oc *ocpackage.OC) ocpackage.NamespacedName {
		t.T().Helper()
		pods := getPods(t, oc, selector, ns)
		if len(pods) == 1 {
			return pods[0]
		} else {
			t.Fatalf("more than one pod found using selector %q in namespace %s: %v", selector, ns, pods)
			panic("should never reach this point because the preceding code either returns or calls t.Fatalf(), which causes a panic")
		}

	}
}

func MatchingSelectorFirst(selector string, ns string) ocpackage.PodLocatorFunc {
	return func(t test.TestHelper, oc *ocpackage.OC) ocpackage.NamespacedName {
		t.T().Helper()
		pods := getPods(t, oc, selector, ns)
		return pods[0]
	}
}

// "placeholder", when we know the name of the pod and the namespace but we want to use a different function which takes PodLocatorFunc as an input parameter
func MatchingName(ns string, name string) ocpackage.PodLocatorFunc {
	return func(t test.TestHelper, oc *ocpackage.OC) ocpackage.NamespacedName {
		return ocpackage.NewNamespacedName(ns, name)
	}
}

package pod

import (
	"strings"

	ocpackage "github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func getPods(t test.TestHelper, oc *ocpackage.OC, selector string, ns string) []ocpackage.NamespacedName {
	t.T().Helper()
	output := oc.Invokef(t, "kubectl -n %s get pods -l %q -o jsonpath='{.items[*].metadata.name}'", ns, selector)
	if output == "" {
		t.Fatalf("no pods found using selector %s in namespace %s", selector, ns)
	}
	pods := strings.Split(output, " ")
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

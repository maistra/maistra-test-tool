package pod

import (
	"strings"

	ocpackage "github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func MatchingSelector(selector string, ns string) ocpackage.PodLocatorFunc {
	return func(t test.TestHelper, oc *ocpackage.OC) ocpackage.NamespacedName {
		t.T().Helper()
		output := oc.Invokef(t, "kubectl -n %s get pods -l %q -o jsonpath='{.items[*].metadata.name}'", ns, selector)
		pods := strings.Split(output, " ")
		switch len(pods) {
		case 0:
			t.Fatalf("no pods found using selector %s in namespace %s", selector, ns)
		case 1:
			return ocpackage.NamespacedName{
				Namespace: ns,
				Name:      pods[0],
			}
		default:
			t.Fatalf("more than one pod found using selector %q in namespace %s: %v", selector, ns, pods)
		}

		panic("should never reach this point because the preceding code either returns or calls t.Fatalf(), which causes a panic")
	}
}

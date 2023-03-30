package pod

import (
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

func MatchingSelector(selector string, ns string) oc.PodLocatorFunc {
	return func(t test.TestHelper) oc.NamespacedName {
		name, err := util.GetPodName(ns, selector)
		if err != nil {
			t.Fatalf("Failed to get pod name: %v", err)
		}
		return oc.NamespacedName{
			Name:      name,
			Namespace: ns,
		}
	}
}

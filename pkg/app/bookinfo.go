package app

import (
	"fmt"
	"net/http/cookiejar"

	"github.com/maistra/maistra-test-tool/pkg/examples"
	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/istio"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

type bookinfo struct {
	ns   string
	mTLS bool
}

var _ App = &bookinfo{}

func Bookinfo(ns string) App {
	return &bookinfo{ns: ns}
}

func BookinfoWithMTLS(ns string) App {
	return &bookinfo{ns: ns, mTLS: true}
}

func (a *bookinfo) Name() string {
	return "bookinfo"
}

func (a *bookinfo) Namespace() string {
	return a.ns
}

func (a *bookinfo) Install(t test.TestHelper) {
	t.T().Helper()

	t.Log("Creating Bookinfo Gateway")
	oc.ApplyFile(t, a.ns, examples.BookinfoGatewayYamlFile())

	t.Log("Creating Bookinfo Destination Rules (all)")
	if a.mTLS {
		oc.ApplyFile(t, a.ns, examples.BookinfoRuleAllMTLSYamlFile())
	} else {
		oc.ApplyFile(t, a.ns, examples.BookinfoRuleAllYamlFile())
	}

	t.Logf("Deploy Bookinfo in namespace %q", a.ns)
	oc.ApplyFile(t, a.ns, examples.BookinfoYamlFile())
}

func (a *bookinfo) Uninstall(t test.TestHelper) {
	t.T().Helper()
	t.Logf("Uninstalling Bookinfo from namespace %q", a.ns)
	oc.DeleteFile(t, a.ns, examples.BookinfoRuleAllYamlFile())
	oc.DeleteFile(t, a.ns, examples.BookinfoGatewayYamlFile())
	oc.DeleteFile(t, a.ns, examples.BookinfoYamlFile())
}

func (a *bookinfo) WaitReady(t test.TestHelper) {
	t.T().Helper()
	oc.WaitDeploymentRolloutComplete(t, a.ns, "productpage-v1", "ratings-v1", "reviews-v1", "reviews-v2", "reviews-v3")
}

func BookinfoLogin(t test.TestHelper, meshNamespace string) *cookiejar.Jar {
	t.T().Helper()

	user := "jason"
	pass := ""
	t.Logf("Logging into bookinfo as %q", user)
	var cookieJar *cookiejar.Jar = nil
	retry.UntilSuccess(t, func(t test.TestHelper) {
		t.T().Helper()
		jar, err := util.SetupCookieJar(user, pass, "http://"+istio.GetIngressGatewayHost(t, meshNamespace))
		if err != nil {
			t.Fatalf("bookinfo login failed: %v", err)
			cookieJar = nil
		}
		cookieJar = jar
	})
	return cookieJar
}

func BookinfoProductPageURL(t test.TestHelper, meshNamespace string) string {
	return fmt.Sprintf("http://%s/productpage", istio.GetIngressGatewayHost(t, meshNamespace))
}

func FindBookinfoProductPageResponseFile(body []byte) string {
	for _, file := range ProductPageResponseFiles {
		if matchesFile(body, file) {
			return file
		}
	}
	return ""
}

func matchesFile(body []byte, file string) bool {
	err := util.CompareHTTPResponse(body, file)
	return err == nil
}

var ProductPageResponseFiles = []string{
	"productpage-normal-user-mongo.html",
	"productpage-normal-user-rating-one-star.html",
	"productpage-normal-user-rating-unavailable.html",
	"productpage-normal-user-v1.html",
	"productpage-normal-user-v2.html",
	"productpage-normal-user-v3.html",
	"productpage-quota-exhausted.html",
	"productpage-rbac-details-reviews-error.html",
	"productpage-rbac-rating-error.html",
	"productpage-review-timeout.html",
	"productpage-test-user-v1.html",
	"productpage-test-user-v2.html",
	"productpage-test-user-v2-rating-unavailable.html",
	"productpage-test-user-v2-review-timeout.html",
}

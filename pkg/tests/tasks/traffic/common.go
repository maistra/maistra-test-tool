package traffic

import (
	"fmt"
	"os"

	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/pod"
	"github.com/maistra/maistra-test-tool/pkg/util/template"

	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

func TestreviewV1(t TestHelper, file string) string {

	ns := "bookinfo"

	reviewV1Podname := pod.MatchingSelector("app=reviews,version=v1", ns)(t, oc.DefaultOC).Name

	templateString, err := os.ReadFile(env.GetRootDir() + "/testdata/resources/html/" + file)
	if err != nil {
		t.Fatalf("could not read template file %s: %v", templateString, err)
	}
	htmlFile := template.Run(t, string(templateString), struct{ ReviewV1Podname string }{ReviewV1Podname: reviewV1Podname})
	fmt.Println(htmlFile)
	os.WriteFile(env.GetRootDir()+"/testdata/resources/html/modified-"+file, []byte(htmlFile), 0644)

	return "modified-" + file

}

func TestreviewV2(t TestHelper, file string) string {

	ns := "bookinfo"

	reviewV2Podname := pod.MatchingSelector("app=reviews,version=v2", ns)(t, oc.DefaultOC).Name
	template2String, err := os.ReadFile(env.GetRootDir() + "/testdata/resources/html/" + file)
	if err != nil {
		t.Fatalf("could not read template file %s: %v", template2String, err)
	}
	html2File := template.Run(t, string(template2String), struct{ ReviewV2Podname string }{ReviewV2Podname: reviewV2Podname})
	os.WriteFile(env.GetRootDir()+"/testdata/resources/html/modified-"+file, []byte(html2File), 0644)

	return "modified-" + file

}

func TestreviewV3(t TestHelper, file string) string {

	ns := "bookinfo"

	reviewV3Podname := pod.MatchingSelector("app=reviews,version=v3", ns)(t, oc.DefaultOC).Name
	template3String, err := os.ReadFile(env.GetRootDir() + "/testdata/resources/html/" + file)
	if err != nil {
		t.Fatalf("could not read template file %s: %v", template3String, err)
	}
	html2File := template.Run(t, string(template3String), struct{ ReviewV3Podname string }{ReviewV3Podname: reviewV3Podname})
	os.WriteFile(env.GetRootDir()+"/testdata/resources/html/modified-"+file, []byte(html2File), 0644)

	return "modified-" + file

}

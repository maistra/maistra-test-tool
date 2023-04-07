package ossm

import (
	_ "embed"
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
	"github.com/maistra/maistra-test-tool/pkg/util/log"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
)

//go:embed yaml/smcp_meta_v2.3.yaml
var smcpV23_template_meta string

func cleanupMultipleSMCP() {
	log.Log.Info("Delete the Multiple CP", meshNamespace)
	util.KubeDeleteContents(meshNamespace, smmr)
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV23_template, Smcp))
	time.Sleep(time.Duration(40) * time.Second)
	util.KubeDeleteContents(meshNamespace, util.RunTemplate(smcpV23_template_meta, Smcp))
	time.Sleep(time.Duration(40) * time.Second)
}

// TestSMCPMutiple tests If multiple SMCPs exist in a namespace, the controller reconciles them all. Jira ticket: https://issues.redhat.com/browse/OSSM-2434
func TestSMCPMutiple(t *testing.T) {
	test.NewTest(t).Id("T36").Groups(test.Full).NotRefactoredYet()
	t.Skip("causes other tests to fail")

	defer cleanupMultipleSMCP()
	defer util.RecoverPanic(t)
	log.Log.Info("Delete Validation Webhook ")
	util.Shell(`oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io`)

	util.ShellMuteOutputError(`oc new-project %s`, meshNamespace)
	util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template, Smcp))
	util.KubeApplyContents(meshNamespace, smmr)
	time.Sleep(time.Duration(20) * time.Second)
	util.KubeApplyContents(meshNamespace, util.RunTemplate(smcpV23_template_meta, Smcp))
	time.Sleep(time.Duration(20) * time.Second)

	log.Log.Info("Verify SMCP status and pods")
	msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNamespace, smcpName)
	if !strings.Contains(msg, "ComponentsReady") {
		log.Log.Error("SMCP not Ready")
		t.Error("SMCP not Ready")
	}

	log.Log.Info("Verify meta control plane and status")
	text, _ := util.Shell(`oc get -n %s smcp/meta -o wide`, meshNamespace)
	if !strings.Contains(text, "ErrMultipleSMCPs") {
		log.Log.Error("SMCP not Ready")
		t.Error("SMCP not Ready")
	}
	util.Shell(`oc get -n %s pods`, meshNamespace)
	util.Shell(`oc wait --for=condition=Ready pods --all -n %s`, meshNamespace)

}

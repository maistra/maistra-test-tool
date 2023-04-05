package ossm

import (
	"strings"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

func cleanupMultipleSMCP() {
	// util.Log.Info("Delete the Multiple CP")
	util.KubeDeleteContents(meshNS2, smmr)
	util.KubeDeleteContents(meshNS2, util.RunTemplate(smcpV23_template, smcp))
	util.KubeDeleteContents(meshNS2, util.RunTemplate(smcpV23_template_meta, smcp))
	util.Shell(`oc -n openshift-operators delete pod -l name=istio-operator`)

}

// TestSMCPMutiple tests If multiple SMCPs exist in a namespace, the controller reconciles them all.
func TestSMCPMutiple(t *testing.T) {
	defer cleanupMultipleSMCP()
	defer util.RecoverPanic(t)
	util.Log.Info("Delete the Validation Webhook ")
	validate_webhook, err := util.Shell(`oc delete validatingwebhookconfiguration/openshift-operators.servicemesh-resources.maistra.io`)
	util.Inspect(err, "Failed to run the command", "", t)
	if strings.Contains(validate_webhook, "deleted") {
		util.Log.Infof("Successfully deleted the validation webhook")
	} else {
		util.Log.Errorf("Failed to delete the validation webhook")
	}

	util.ShellMuteOutputError(`oc new-project %s`, meshNS2)
	util.KubeApplyContents(meshNS2, util.RunTemplate(smcpV23_template, smcp))
	util.KubeApplyContents(meshNS2, smmr)
	time.Sleep(time.Duration(20) * time.Second)
	util.KubeApplyContents(meshNS2, util.RunTemplate(smcpV23_template_meta, smcp))
	time.Sleep(time.Duration(20) * time.Second)

	util.Log.Info("Verify SMCP status and pods")
	msg, _ := util.Shell(`oc get -n %s smcp/%s -o wide`, meshNS2, smcpName)
	if !strings.Contains(msg, "ComponentsReady") {
		util.Log.Error("SMCP not Ready")
		t.Error("SMCP not Ready")
	}

	util.Log.Info("Verify meta control plane and status")
	text, _ := util.Shell(`oc get -n %s smcp/meta -o wide`, meshNS2)
	if !strings.Contains(text, "ErrMultipleSMCPs") {
		util.Log.Error("SMCP not Ready")
		t.Error("SMCP not Ready")
	}
	util.Shell(`oc get -n %s pods`, meshNS2)
	util.Shell(`oc wait --for=condition=Ready pods --all -n %s`, meshNS2)

}

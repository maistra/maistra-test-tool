package ossm

import (
	"strings"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util"
)

// referance - https://issues.redhat.com/browse/OSSM-2374

// testing Jira - https://issues.redhat.com/browse/OSSM-3450

func TestSMMDelete(t *testing.T) {

	defer util.RecoverPanic(t)
	util.Log.Info("Delete Service Mesh Member ")
	util.DeleteSMMR(meshNamespace)

	util.Log.Info("Create SMM in ns1")
	util.KubeApplyContents("bookinfo", ServiceMeshMember_1)

	util.Log.Info("Create SMM in ns2")
	util.KubeApplyContents("my-awesome-project", ServiceMeshMember_2)

	util.Log.Info("Verify the SMMR will create Automatically with SMM")
	smmrVerification, err := util.Shell("oc wait --for condition=Ready -n %s smmr/default --timeout 20s", meshNamespace)
	util.Inspect(err, "Failed to run the command", "", t)
	if strings.Contains(smmrVerification, "condition met") {
		util.Log.Infof("Success, smmr installed successfully with smm")
	} else {
		util.Log.Errorf("Failed, smmr not installed with smm")
	}

	util.Log.Info("Delete SMM in ns2")
	util.DeleteSMM("my-awesome-project")

	util.Log.Info("Verify the SMMR still available for SMM1")
	smmrVerification1, err := util.Shell("oc wait --for condition=Ready -n %s smmr/default --timeout 20s", meshNamespace)
	util.Inspect(err, "Failed to run the command", "", t)
	if strings.Contains(smmrVerification1, "condition met") {
		util.Log.Infof("Success, smmr installed successfully with smm")
	} else {
		util.Log.Errorf("Failed, smmr not installed with smm")
	}

	util.Log.Info("Delete SMM in ns1")
	util.DeleteSMM("bookinfo")

	util.Log.Info("Verify the all the SMM deleted then SMMR will delete automatically")
	smmrVerification2, err := util.Shell("oc get smmr -n istio-system")
	util.Inspect(err, "Failed to run the command", "", t)
	if strings.Contains(smmrVerification2, "No resources found") {
		util.Log.Infof("Success, smmr uninstalled successfully with smm")
	} else {
		util.Log.Errorf("Failed, smmr not uninstalled with smm")
	}
}

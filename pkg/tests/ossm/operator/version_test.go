package operator

func TestVersion(t *testing.T) {
	NewTest(t).Run(func(t TestHelper) {
		t.Log("Test to verify helm chart version matches expected version")

		operatorPod := pod.MatchingSelector("name=istio-operator", env.GetOperatorNamespace())
		
		cmd := exec.Command("kubectl exec deploy/istio-operator -n " + env.GetOperatorNamespace() +
			" -- cat /usr/local/share/istio-operator/helm/" + env.GetSMCPVersion() +
			"/istio-control/istio-discovery/templates/deployment.yaml | grep maistra-version | awk '{print $2}'")
		
			outputBytes, err := cmd.Output()
		if err != nil{
			t.Fatalf("Failed to execture command: %v", err)
		}

		output := string(outputBytes)
		
	},
			
		//kubectl exec deploy/istio-operator -- cat /usr/local/share/istio-operator/helm/v2.5/istio-control/istio-discovery/templates/deployment.yaml | grep maistra-version
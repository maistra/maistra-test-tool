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

package non_dependant

import (
	_ "embed"
	"testing"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/tests/ossm"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

func TestOlmWebhookCreation(t *testing.T) {
	NewTest(t).Groups(Full, ARM, Disconnected).Run(func(t TestHelper) {
		t.Log("This test verifies that OLM creates all validating/mutating webhooks")
		t.Log("See https://issues.redhat.com/browse/OSSM-6762")
		if env.GetOperatorVersion().LessThan(version.OPERATOR_2_6_0) {
			t.Skip("Skipping until 2.6 operator")
		}

		t.Cleanup(func() {
			t.LogStepf("Delete namespace %s", meshNamespace)
			oc.RecreateNamespace(t, meshNamespace)
		})

		t.LogStepf("Delete and recreate namespace %s", meshNamespace)
		oc.RecreateNamespace(t, meshNamespace)

		t.NewSubTest("Check global webhooks").Run(func(t TestHelper) {
			t.Log("Check that global validatingwebhookconfiguration's were created by OLM")
			checkGlobalWebhooks(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smcp.validation.maistra.io")
			checkGlobalWebhooks(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smmr.validation.maistra.io")
			checkGlobalWebhooks(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smm.validation.maistra.io")

			t.Log("Check that global validatingwebhookconfiguration's were recreated by OLM after deletion")
			deleteGlobalWebhook(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smcp.validation.maistra.io")
			deleteGlobalWebhook(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smmr.validation.maistra.io")
			deleteGlobalWebhook(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smm.validation.maistra.io")
			retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(30).DelayBetweenAttempts(5*time.Second), func(t TestHelper) {
				checkGlobalWebhooks(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smcp.validation.maistra.io")
				checkGlobalWebhooks(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smmr.validation.maistra.io")
				checkGlobalWebhooks(t, "validatingwebhookconfiguration", "olm.webhook-description-generate-name=smm.validation.maistra.io")
			})

			t.Log("Check that global mutatingwebhookconfiguration's were created by OLM")
			checkGlobalWebhooks(t, "mutatingwebhookconfiguration", "olm.webhook-description-generate-name=smcp.mutation.maistra.io")
			checkGlobalWebhooks(t, "mutatingwebhookconfiguration", "olm.webhook-description-generate-name=smmr.mutation.maistra.io")

			t.Log("Check that global mutatingwebhookconfiguration's were recreated by OLM after deletion")
			deleteGlobalWebhook(t, "mutatingwebhookconfiguration", "olm.webhook-description-generate-name=smcp.mutation.maistra.io")
			deleteGlobalWebhook(t, "mutatingwebhookconfiguration", "olm.webhook-description-generate-name=smmr.mutation.maistra.io")
			retry.UntilSuccessWithOptions(t, retry.Options().MaxAttempts(30).DelayBetweenAttempts(5*time.Second), func(t TestHelper) {
				checkGlobalWebhooks(t, "mutatingwebhookconfiguration", "olm.webhook-description-generate-name=smcp.mutation.maistra.io")
				checkGlobalWebhooks(t, "mutatingwebhookconfiguration", "olm.webhook-description-generate-name=smmr.mutation.maistra.io")
			})
		})

		t.NewSubTest("Check smcp related webhooks").Run(func(t TestHelper) {
			t.Log("Check that smcp related webhooks doesn't exist")
			checkSmcpWebhooksDoesNotExist(t, "validatingwebhookconfiguration", "maistra.io/owner-name="+env.GetDefaultSMCPName())
			checkSmcpWebhooksDoesNotExist(t, "mutatingwebhookconfiguration", "maistra.io/owner-name="+env.GetDefaultSMCPName())

			t.LogStep("Create the SMCP")
			ossm.DeployControlPlane(t)

			t.Log("Check that smcp related webhooks were created by OLM")
			checkSmcpWebhooksExist(t, "validatingwebhookconfiguration", "maistra.io/owner-name="+env.GetDefaultSMCPName())
			checkSmcpWebhooksExist(t, "mutatingwebhookconfiguration", "maistra.io/owner-name="+env.GetDefaultSMCPName())

			t.LogStep("Delete the SMCP")
			oc.RecreateNamespace(t, meshNamespace)

			t.Log("Check that smcp related webhooks were created by OLM")
			checkSmcpWebhooksDoesNotExist(t, "validatingwebhookconfiguration", "maistra.io/owner-name="+env.GetDefaultSMCPName())
			checkSmcpWebhooksDoesNotExist(t, "mutatingwebhookconfiguration", "maistra.io/owner-name="+env.GetDefaultSMCPName())
		})
	})
}

func checkGlobalWebhooks(t TestHelper, kind string, label string) {
	if oc.ResourceByLabelExists(t, "", kind, label) {
		t.LogSuccessf("Got the expected %s with label %s", kind, label)
	} else {
		t.Fatalf("Expect to find %s with label %s created automatically by OLM", kind, label)
	}
}

func checkSmcpWebhooksExist(t TestHelper, kind string, label string) {
	retry.UntilSuccess(t, func(t TestHelper) {
		t.Logf("Check that smcp %s was created by OLM", kind)
		if oc.ResourceByLabelExists(t, "", kind, label) {
			t.LogSuccessf("Got the expected %s with label %s", kind, label)
		} else {
			t.Fatalf("Expect to find %s with label %s for smcp", kind, label)
		}
	})
}

func checkSmcpWebhooksDoesNotExist(t TestHelper, kind string, label string) {
	retry.UntilSuccess(t, func(t TestHelper) {
		t.Logf("Check that smcp %s was deleted by OLM", kind)
		if oc.ResourceByLabelExists(t, "", kind, label) {
			t.Fatalf("Expect to not find %s with label %s for smcp but it was found", kind, label)
		} else {
			t.LogSuccessf("Expect to not find %s with label %s for smcp", kind, label)
		}
	})
}

func deleteGlobalWebhook(t TestHelper, kind string, label string) {
	name := oc.GetAllResoucesNamesByLabel(t, "", kind, label)[0]
	oc.DeleteResource(t, "", kind, name)
}

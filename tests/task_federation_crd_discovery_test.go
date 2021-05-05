// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"fmt"
	"io/ioutil"
	"maistra/util"
	"testing"
	"time"

	"istio.io/pkg/log"
)

func cleanupTestFedDiscovery() {
	log.Info("# Cleanup ...")
	cleanBookinfo("mesh1-bookinfo")
	cleanBookinfo("mesh2-bookinfo")

}

func TestFedDiscovery(t *testing.T) {
	defer cleanupTestFedDiscovery()

	log.Info("Running setup install.sh")
	util.Shell(`pushd ../pkg/federation/test; ./install.sh; popd;`)
	deployBookinfo("mesh1-bookinfo", false)
	deployBookinfo("mesh2-bookinfo", false)
	productpageURL := fmt.Sprintf("http://%s/productpage", gatewayHTTP)

	t.Run("federation_crd_redirect_ratings", func(t *testing.T) {
		defer cleanupTestFedDiscovery()

		if err := util.KubeApply("mesh2-bookinfo", "", kubeconfig); err != nil {
			t.Errorf("Failed to update VirtualService")
			log.Errorf("Failed to update VirtualService")
		}
		time.Sleep(time.Duration(waitTime) * time.Second)

		for i := 0; i <= 5; i++ {
			resp, duration, err := util.GetHTTPResponse(productpageURL, nil)
			util.Inspect(err, "Failed to get HTTP Response", "", t)
			log.Infof("bookinfo productpage returned in %d ms", duration)
			defer util.CloseResponseBody(resp)
			body, err := ioutil.ReadAll(resp.Body)
			util.Inspect(err, "Failed to read response body", "", t)
			util.Inspect(
				util.CompareHTTPResponse(body, "productpage-normal-user-v1.html"),
				"Didn't get expected response.",
				"Success. Routing traffic to all v1.",
				t)
		}
	})
}

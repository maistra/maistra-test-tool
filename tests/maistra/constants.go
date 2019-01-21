// Copyright 2019 Istio Authors
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

// Package dashboard provides testing of the grafana dashboards used in Istio
// to provide mesh monitoring capabilities.

package maistra

const (
	modelDir					= "testdata/modelDir"
	bookinfoAllv1Yaml			= "testdata/bookinfo/networking/virtual-service-all-v1.yaml"
	bookinfoReviewTestv2Yaml	= "testdata/bookinfo/networking/virtual-service-reviews-test-v2.yaml"
	bookinfoRatingDelayYaml		= "testdata/bookinfo/networking/virtual-service-ratings-test-delay.yaml"
	bookinfoRatingDelayv2Yaml	= "testdata/bookinfo/networking/virtual-service-ratings-test-delay-2.yaml"
	bookinfoRatingAbortYaml		= "testdata/bookinfo/networking/virtual-service-ratings-test-abort.yaml"
	bookinfoReview50v3Yaml 		= "testdata/bookinfo/networking/virtual-service-reviews-50-v3.yaml"
	bookinfoReviewv3Yaml 		= "testdata/bookinfo/networking/virtual-service-reviews-v3.yaml"
	testUsername				= "jason"
	
)
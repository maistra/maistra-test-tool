// Copyright 2019 Red Hat, Inc.
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

package maistra

import (
	"io/ioutil"
	"strings"
)

var (
	bookinfoRBACOn string
	bookinfoRBAConDB string
	bookinfoNamespacePolicy string
	bookinfoProductpagePolicy string
	bookinfoReviewPolicy string
	bookinfoRatingPolicy string
	bookinfoMongodbPolicy string
)

func updateYaml(namespace string) {
	data, _ := ioutil.ReadFile(bookinfoRBACOnTemplate)
	bookinfoRBACOn = strings.Replace(string(data), "\"default\"", "\"" + namespace + "\"", -1)

	data, _ = ioutil.ReadFile(bookinfoRBACOnDBTemplate)
	bookinfoRBAConDB = strings.Replace(string(data), "mongodb.default", "mongodb." + namespace, -1)

	data, _ = ioutil.ReadFile(bookinfoNamespacePolicyTemplate)
	bookinfoNamespacePolicy = strings.Replace(string(data), "default", namespace, -1)

	data, _ = ioutil.ReadFile(bookinfoProductpagePolicyTemplate)
	bookinfoProductpagePolicy = strings.Replace(string(data), "default", namespace, -1)

	data, _ = ioutil.ReadFile(bookinfoReviewPolicyTemplate)
	bookinfoReviewPolicy = strings.Replace(string(data), "default", namespace, -1)

	data, _ = ioutil.ReadFile(bookinfoRatingPolicyTemplate)
	bookinfoRatingPolicy = strings.Replace(string(data), "default", namespace, -1)

	data, _ = ioutil.ReadFile(bookinfoMongodbPolicyTemplate)
	bookinfoMongodbPolicy = strings.Replace(string(data), "default", namespace, -1)

}
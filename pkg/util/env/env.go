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

package env

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/maistra/maistra-test-tool/pkg/util/version"
)

var initTime = time.Now()

// getenv returns an environment variable value or the given fallback as a default value.
func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

// GetRootDir gets the project root dir from the current working directory (which is usually the current test's package dir)
func GetRootDir() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	index := strings.LastIndex(dir, "/pkg/tests/")
	if index == -1 {
		panic("expected working dir to be a subdir of .../pkg/tests/, but was " + dir)
	}
	return dir[:index]
}

func IsRosa() bool {
	return getenv("ROSA", "false") == "true"
}

func IsNightly() bool {
	return getenv("NIGHTLY", "false") == "true"
}

func GetDefaultSMCPName() string {
	return getenv("SMCP_NAME", "basic")
}

func GetDefaultMeshNamespace() string {
	return getenv("SMCP_NAMESPACE", "istio-system")
}

func GetSMCPVersion() version.Version {
	return version.ParseVersion(getenv("SMCP_VERSION", "v2.6"))
}

func GetOperatorVersion() version.Version {
	return version.ParseVersion(getenv("OPERATOR_VERSION", "2.6.0"))
}

func GetArch() string {
	return getenv("OCP_ARCH", "x86")
}

func GetTestGroup() string {
	return getenv("TEST_GROUP", "full")
}

func GetMustGatherImage() string {
	return getenv("MUST_GATHER_IMAGE", "registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel8:"+GetMustGatherTag())
}

func GetMustGatherTag() string {
	return getenv("MUST_GATHER_TAG", fmt.Sprintf("%d.%d", GetOperatorVersion().Major, GetOperatorVersion().Minor))
}

func IsMustGatherEnabled() bool {
	return getenv("MUST_GATHER", "true") == "true"
}

func GetKubeconfig() string {
	return getenv("KUBECONFIG", "")
}

func GetKubeconfig2() string {
	return getenv("KUBECONFIG2", "")
}

func GetOperatorNamespace() string {
	return "openshift-operators"
}

func IsLogFailedRetryAttempts() bool {
	return getenv("LOG_FAILED_RETRY_ATTEMPTS", "true") == "true"
}

func GetOutputDir() string {
	return getenv("OUTPUT_DIR", fmt.Sprintf("%s/tests/result-%s/%s", GetRootDir(), initTime.Format("20060102150405"), GetSMCPVersion()))
}

func IsMetalLBInternalIPEnabled() bool {
	return getenv("METALLB_INTERNAL_IP_ENABLED", "false") == "true"
}

func GetExpectedVersion() string {
	return getenv("EXPECTED_VERSION", "")
}

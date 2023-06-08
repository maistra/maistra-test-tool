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
	return version.ParseVersion(getenv("SMCP_VERSION", "v2.4"))
}

func GetArch() string {
	return getenv("OCP_ARCH", "x86")
}

func GetTestGroup() string {
	return getenv("TEST_GROUP", "full")
}

func GetMustGatherImage() string {
	return "registry.redhat.io/openshift-service-mesh/istio-must-gather-rhel8:" + getenv("MUST_GATHER_TAG", "2.3")
}

func GetMustGatherTag() string {
	return getenv("MUST_GATHER_TAG", "2.3")
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

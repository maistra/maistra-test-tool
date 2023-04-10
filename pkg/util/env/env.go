package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

var initEnvVarsOnce sync.Once

// getenv loads test.env file and returns an environment variable value.
// If the environment variable is empty, it returns the fallback as a default value.
func Getenv(key, fallback string) string {
	InitEnvVarsFromFile()
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func GetenvAsInt(key string, fallback int) int {
	value := Getenv(key, strconv.Itoa(fallback))
	num, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("env var %s must be an integer, but was: %s", key, value))
	}
	return num
}

func InitEnvVarsFromFile() {
	initEnvVarsOnce.Do(func() {
		envFilePath := GetRootDir() + "/tests/test.env"
		if err := godotenv.Load(envFilePath); err != nil {
			panic(fmt.Sprintf("Error loading file %s", envFilePath))
		}
	})
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
	return Getenv("ROSA", "false") == "true"
}

func GetDefaultSMCPName() string {
	return Getenv("SMCPNAME", "basic")
}

func GetDefaultMeshNamespace() string {
	return Getenv("MESHNAMESPACE", "istio-system")
}

func GetDefaultSMCPVersion() string {
	return Getenv("SMCPVERSION", "2.4")
}

func GetOperatorNamespace() string {
	return "openshift-operators"
}

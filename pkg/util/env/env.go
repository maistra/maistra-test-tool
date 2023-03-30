package env

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"

	"github.com/maistra/maistra-test-tool/pkg/util/log"
)

// getenv loads test.env file and returns an environment variable value.
// If the environment variable is empty, it returns the fallback as a default value.
func Getenv(key, fallback string) string {
	if err := godotenv.Load("test.env"); err != nil {
		log.Log.Fatal("Error loading .env file")
	}
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

package env

import (
	"os"

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

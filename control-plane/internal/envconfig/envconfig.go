// Package envconfig provides small helpers for reading environment
// variables at process startup.
package envconfig

import (
	"log"
	"os"
	"strconv"
)

// MustEnv exits the process if name is unset.
func MustEnv(name string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		log.Fatalf("missing required environment variable %s", name)
	}
	return value
}

// MustEnvInt is MustEnv parsed as an integer; exits the process if it
// isn't one.
func MustEnvInt(name string) int {
	value, err := strconv.Atoi(MustEnv(name))
	if err != nil {
		log.Fatalf("environment variable %s must be an integer: %v", name, err)
	}
	return value
}

func EnvOr(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return value
	}
	return fallback
}

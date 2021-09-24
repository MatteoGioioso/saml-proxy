package sharedKernel

import (
	"github.com/crewjam/saml"
	"io"
	"os"
	"strconv"
)

func GetEnvWithFallbackBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		v, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}
		return v
	}

	return fallback
}

func GetEnvWithFallbackString(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func RandomBytes(n int) []byte {
	rv := make([]byte, n)

	if _, err := io.ReadFull(saml.RandReader, rv); err != nil {
		panic(err)
	}
	return rv
}

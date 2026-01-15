//go:build ionos
// +build ionos

package main

import (
	"fmt"
	"os"

	"github.com/libdns/ionos"
)

func GetProvider() (Provider, string, error) {
	identifier := "ionos"
	envVar := "DDNS_IONOS_APITOKEN"
	value := os.Getenv(envVar)
	if value == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	return &ionos.Provider{
		AuthAPIToken: value,
	}, identifier, nil
}

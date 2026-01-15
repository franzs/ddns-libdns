//go:build desec
// +build desec

package main

import (
	"fmt"
	"os"

	"github.com/libdns/desec"
)

func GetProvider() (Provider, string, error) {
	identifier := "deSEC"
	envVar := "DDNS_DESEC_TOKEN"
	value := os.Getenv(envVar)
	if value == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	return &desec.Provider{
		Token: value,
	}, identifier, nil
}

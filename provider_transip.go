//go:build transip
// +build transip

package main

import (
	"fmt"
	"os"

	"github.com/libdns/transip"
)

func GetProvider() (Provider, string, error) {
	identifier := "transip"
	envVar := "DDNS_TRANSIP_PRIVATEKEY"
	value := os.Getenv(envVar)
	if value == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	envVar = "DDNS_TRANSIP_USER"
	user := os.Getenv(envVar)
	if user == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	return &transip.Provider{
		AuthLogin:  user,
		PrivateKey: value,
	}, identifier, nil
}

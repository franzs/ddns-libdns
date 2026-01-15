//go:build hetzner
// +build hetzner

package main

import (
	"fmt"
	"os"

	"github.com/libdns/hetzner/v2"
)

func GetProvider() (Provider, string, error) {
	identifier := "hetzner"
	envVar := "DDNS_HETZNER_TOKEN"
	value := os.Getenv(envVar)
	if value == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	return &hetzner.Provider{
		APIToken: value,
	}, identifier, nil
}

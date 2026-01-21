//go:build cloudflare
// +build cloudflare

package main

import (
	"fmt"
	"os"

	"github.com/libdns/cloudflare"
)

func GetProvider() (Provider, string, error) {
	identifier := "cloudflare"
	envVar := "DDNS_CLOUDFLARE_APITOKEN"
	value := os.Getenv(envVar)
	if value == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	return &cloudflare.Provider{
		APIToken: value,
	}, identifier, nil
}

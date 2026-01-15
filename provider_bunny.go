//go:build bunny
// +build bunny

package main

import (
	"fmt"
	"os"

	"github.com/libdns/bunny"
)

func GetProvider() (Provider, string, error) {
	identifier := "bunny"
	envVar := "DDNS_BUNNY_ACCESSKEY"
	value := os.Getenv(envVar)
	if value == "" {
		return nil, "", fmt.Errorf("%s environment variable is required", envVar)
	}

	return &bunny.Provider{
		AccessKey: value,
	}, identifier, nil
}

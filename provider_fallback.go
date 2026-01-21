//go:build !bunny && !cloudflare && !desec && !hetzner && !ionos
// +build !bunny,!cloudflare,!desec,!hetzner,!ionos

package main

import "fmt"

// GetProvider is the fallback implementation that gets compiled
// if no specific provider build tag (like 'bunny' or 'cloudflare') is used.
func GetProvider() (Provider, string, error) {
	return nil, "", fmt.Errorf("no DNS provider implementation found for this build. " +
		"Please build with a specific tag, e.g., 'go build -tags=bunny' or specify DDNS_PROVIDER_TAG in Dockerfile.")
}

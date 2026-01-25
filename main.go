package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-crypt/crypt"
	"github.com/libdns/libdns"
)

// Config represents various configuration variables
type Config struct {
	APIKey   string
	AuthJSON string
	Port     string
	TTL      int
}

// UserConfig represents the JSON structure from the environment variable
type UserConfig struct {
	Username     string   `json:"username"`
	PasswordHash string   `json:"passwordHash"`
	Hostnames    []string `json:"hostnames"`
}

// User represents the internal lookup structure
type User struct {
	PasswordHash string
	AllowedHosts map[string]bool // Set for O(1) lookup
}

type Provider interface {
	SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error)
	ListZones(ctx context.Context) ([]libdns.Zone, error)
}

var (
	authUsers map[string]User // Map[Username] -> User Struct
	config    Config
	provider  Provider
)

func loadAuthUsers() error {
	var rawUsers []UserConfig

	if err := json.Unmarshal([]byte(config.AuthJSON), &rawUsers); err != nil {
		return fmt.Errorf("Failed to parse JSON from DDNS_AUTH_CONFIG: %v", err)
	}

	authUsers = make(map[string]User)

	for _, ru := range rawUsers {
		var allowsHostnames []string

		allowed := make(map[string]bool)
		for _, h := range ru.Hostnames {
			// Normalize: lowercase and trim trailing dots
			normalized := strings.ToLower(strings.TrimSuffix(h, "."))
			allowed[normalized] = true
			allowsHostnames = append(allowsHostnames, normalized)
		}
		authUsers[ru.Username] = User{
			PasswordHash: ru.PasswordHash,
			AllowedHosts: allowed,
		}
		slog.Info("Loaded user",
			"username", ru.Username,
			"hostnames", allowsHostnames)
	}

	return nil
}

func inferZoneAndName(ctx context.Context, hostname string) (string, string, error) {
	zones, err := provider.ListZones(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to list zones: %w", err)
	}

	for _, z := range zones {
		recordName := libdns.RelativeName(hostname, z.Name)
		if recordName != hostname {
			return z.Name, recordName, nil
		}
	}

	return "", "", fmt.Errorf("Can't find zone from hostname %s", hostname)
}

func updateDNS(ctx context.Context, zone, recordName string, ipaddrs []netip.Addr) error {
	var recordsToSet []libdns.Record

	for _, ip := range ipaddrs {
		recordsToSet = append(recordsToSet, libdns.Record(
			libdns.Address{
				Name: recordName,
				IP:   ip,
				TTL:  time.Duration(config.TTL) * time.Second,
			},
		))
	}

	// SetRecords updates existing records or creates them if missing
	// It returns the records that were set.
	_, err := provider.SetRecords(ctx, zone, recordsToSet)
	return err
}

func handleUpdate(w http.ResponseWriter, r *http.Request) {
	var parsedIPAddrs []netip.Addr

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// --- Authentication & Authorization ---

	// Check HTTP Basic Auth
	username, password, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		slog.Error("Can't decode username and password using BasicAuth(). No credentials?")
		http.Error(w, "badauth", http.StatusUnauthorized)
		return
	}

	// Verify Credentials
	user, exists := authUsers[username]
	if !exists {
		// Use a dummy hash to maintain consistent timing
		user = User{PasswordHash: "$argon2id$v=19$m=4096,t=3,p=1$WThNMStEazRDM3NVQkIxOXlVaHRaQT09$3XjzaHozsLfjY3ejWY91y7sQ964r49uBsB15PZWVOGw"}
	}

	valid, err := crypt.CheckPassword(password, user.PasswordHash)
	if !exists || !valid || err != nil {
		// Differentiate slightly for logs, but return Generic 401 to client
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)

		if !exists {
			slog.Error("Auth Failed: User does not exist", "username", username)
		} else if err != nil {
			slog.Error("Auth Failed: Can't CheckPassword()", "username", username, "error", err.Error())
		} else if !valid {
			slog.Error("Auth Failed: Invalid password", "username", username)
		}

		http.Error(w, "badauth", http.StatusUnauthorized)
		return
	}

	// Validate Parameters
	rawHostname := r.URL.Query().Get("hostname")
	if rawHostname == "" {
		slog.Error("No hostname provided")
		http.Error(w, "notfqdn", http.StatusBadRequest)
		return
	}

	// Normalize hostname for checking against the whitelist
	normalizedHostname := strings.ToLower(strings.TrimSuffix(rawHostname, "."))

	// Check Authorization (ACL)
	if !user.AllowedHosts[normalizedHostname] {
		slog.Error("Access Denied: User tried to update forbidden host", "username", username, "hostname", normalizedHostname)
		http.Error(w, "badauth", http.StatusForbidden)
		return
	}

	// --- DNS Update Logic ---

	myip := r.URL.Query().Get("myip")

	if myip == "" {
		slog.Error("No ip address provided")
		http.Error(w, "badrequest", http.StatusBadRequest)
		return
	}

	for _, ip := range strings.Split(myip, ",") {
		parsedIPAddr, err := netip.ParseAddr(ip)
		if err != nil {
			slog.Error("Unable to parse ip address using netip.ParseAddr()", "ipaddr", ip)
			http.Error(w, "badrequest", http.StatusBadRequest)
			return
		}

		// Skip 0.0.0.0, ::, ::0, etc.
		if parsedIPAddr.IsUnspecified() {
			continue
		}

		parsedIPAddrs = append(parsedIPAddrs, parsedIPAddr)
	}

	if len(parsedIPAddrs) == 0 {
		slog.Error("No specified IP address given", "myip", myip)
		http.Error(w, "badrequest", http.StatusBadRequest)
		return
	}

	zone, recordName, err := inferZoneAndName(ctx, normalizedHostname)
	if err != nil {
		slog.Error(err.Error())
		http.Error(w, "dnserr", http.StatusBadRequest)
		return
	}

	if err := updateDNS(ctx, zone, recordName, parsedIPAddrs); err != nil {
		slog.Error(err.Error())
		http.Error(w, "dnserr", http.StatusInternalServerError)
		return
	}

	slog.Info("Successful update",
		"username", username,
		"hostname", normalizedHostname,
		"zone", zone,
		"recordname", recordName,
		"ipaddr", parsedIPAddrs)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("good " + myip))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Optionally check provider connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if _, err := provider.ListZones(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unhealthy: " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func main() {
	var (
		identifier string
		err        error
	)

	// Setup logger to output JSON to Stdout
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Set as default
	slog.SetDefault(logger)

	// Load Configuration
	config = Config{
		AuthJSON: os.Getenv("DDNS_AUTH_CONFIG"),
		Port:     os.Getenv("DDNS_PORT"),
	}

	if config.AuthJSON == "" {
		slog.Error("DDNS_AUTH_CONFIG environment variable is required")
		os.Exit(1)
	}
	if config.Port == "" {
		config.Port = "8080"
	}

	ttlEnv := os.Getenv("DDNS_TTL")
	if ttlEnv == "" {
		config.TTL = 60
	} else {
		config.TTL, err = strconv.Atoi(ttlEnv)
		if err != nil {
			slog.Error(fmt.Sprintf("Can't parse DDNS_TTL environment variable: %v", err))
			os.Exit(1)
		}

		if config.TTL < 60 || config.TTL > 86400 {
			slog.Error("DDNS_TTL must be between 60 and 86400 seconds")
			os.Exit(1)
		}
	}

	// Load and Parse Auth Config
	if err = loadAuthUsers(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	provider, identifier, err = GetProvider()
	if err != nil {
		slog.Error(fmt.Sprintf("Error initializing provider: %v", err))
		os.Exit(1)
	}

	// Start Server
	http.HandleFunc("/v3/update", handleUpdate)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/ready", handleHealth)

	slog.Info("Starting ddns-libdns", "port", config.Port, "provider", identifier)
	server := &http.Server{
		Addr:         ":" + config.Port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error(err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
}

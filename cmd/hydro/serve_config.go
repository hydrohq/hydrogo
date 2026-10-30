package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/tenant"
)

// serveOptions are the serve-only command-line switches; everything else
// comes from config.Config.
type serveOptions struct {
	cgi            bool
	fastCGI        bool
	fastCGIAddress string
	dev            bool
	runtime        runtimeFlags
}

// parseServeArgs parses the serve flags and loads the runtime configuration,
// applying the standalone HTTPS switches.
func parseServeArgs(args []string, stderr io.Writer) (serveOptions, config.Config, error) {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	cgiMode := flags.Bool("cgi", false, "handle one request from a CGI web server")
	fastCGIMode := flags.Bool("fastcgi", false, "serve FastCGI requests (via standard input or listener)")
	fastCGIAddress := flags.String("fastcgi-address", "", "FastCGI listener address (loopback like 127.0.0.1:9001; if empty, uses standard input / mod_fcgid)")
	sslMode := flags.String("ssl", "", "enable standalone HTTPS with --ssl=auto")
	sslDomain := flags.String("domain", "", "hostname for standalone HTTPS")
	devFlag := flags.Bool("dev", false, "development mode: relaxed headers, demo frontend and Redis fallback; requires a loopback --address")
	values := addRuntimeFlags(flags)
	if err := flags.Parse(args); err != nil {
		return serveOptions{}, config.Config{}, err
	}
	if flags.NArg() != 0 {
		return serveOptions{}, config.Config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *cgiMode && *fastCGIMode {
		return serveOptions{}, config.Config{}, errors.New("--cgi and --fastcgi cannot be combined")
	}
	opts := serveOptions{cgi: *cgiMode, fastCGI: *fastCGIMode, fastCGIAddress: *fastCGIAddress, dev: *devFlag, runtime: values}

	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil {
		return serveOptions{}, config.Config{}, err
	}
	if *sslMode != "" {
		if *sslMode != "auto" || strings.TrimSpace(*sslDomain) == "" {
			return serveOptions{}, config.Config{}, errors.New("--ssl=auto requires --domain")
		}
		if cfg.HTTP.BehindProxy {
			return serveOptions{}, config.Config{}, errors.New("standalone automatic TLS cannot run behind a reverse proxy")
		}
		cfg.HTTP.AutoTLS, cfg.HTTP.Domain = true, strings.TrimSpace(*sslDomain)
		if cfg.HTTP.PublicOrigin == "" {
			cfg.HTTP.PublicOrigin = "https://" + cfg.HTTP.Domain
		}
		cfg.Security.EnableHSTS = true
		cfg.Security.SecureCookies = true
	}
	if cfg.HTTP.AutoTLS {
		if cfg.HTTP.PublicOrigin == "" {
			cfg.HTTP.PublicOrigin = "https://" + cfg.HTTP.Domain
		}
		cfg.Security.EnableHSTS, cfg.Security.SecureCookies = true, true
		if err := cfg.Validate(); err != nil {
			return serveOptions{}, config.Config{}, err
		}
	}
	return opts, cfg, nil
}

// configureTenancy builds the host resolver and the sorted list of tenant
// scopes when tenancy is enabled.
func configureTenancy(cfg config.Config) (*tenant.HostResolver, []tenant.Scope, error) {
	var tenantResolver *tenant.HostResolver
	var tenantScopes []tenant.Scope
	if cfg.Tenancy.Enabled {
		hostScopes := make(map[string]tenant.Scope, len(cfg.Tenancy.Hosts))
		hostNames := make([]string, 0, len(cfg.Tenancy.Hosts))
		for host := range cfg.Tenancy.Hosts {
			hostNames = append(hostNames, host)
		}
		sort.Strings(hostNames)
		seenTenants := make(map[string]bool)
		for _, host := range hostNames {
			tenantID := cfg.Tenancy.Hosts[host]
			scope, scopeErr := tenant.NewScope(tenantID, tenantID)
			if scopeErr != nil {
				return nil, nil, fmt.Errorf("tenancy host %q: %w", host, scopeErr)
			}
			hostScopes[host] = scope
			if !seenTenants[tenantID] {
				seenTenants[tenantID] = true
				tenantScopes = append(tenantScopes, scope)
			}
		}
		resolver := tenant.NewHostResolver(hostScopes, cfg.Tenancy.RequireHost)
		tenantResolver = &resolver
	}
	return tenantResolver, tenantScopes, nil
}

// enableDevMode turns on development mode for --dev or HYDRO_DEV_MODE=1. The
// returned restore function unsets HYDRO_DEV_MODE again when serve set it.
func enableDevMode(address string, devFlag bool, stderr io.Writer) (bool, func(), error) {
	restore := func() {}
	devModeEnv := strings.TrimSpace(os.Getenv("HYDRO_DEV_MODE")) == "1"
	devMode := devFlag || devModeEnv
	if devMode {
		if !devModeEnv {
			// Other packages read HYDRO_DEV_MODE; scope it to this serve call.
			_ = os.Setenv("HYDRO_DEV_MODE", "1")
			restore = func() { _ = os.Unsetenv("HYDRO_DEV_MODE") }
		}
		if err := enforceDevModeLocalBind(address); err != nil {
			restore()
			return false, func() {}, err
		}
		printDevModeWarning(stderr)
	}
	return devMode, restore, nil
}

// startDebugPprof serves net/http/pprof on HYDRO_DEBUG_PPROF_ADDR (loopback
// only) and returns a function that stops it.
func startDebugPprof() (func(), error) {
	addr := strings.TrimSpace(os.Getenv("HYDRO_DEBUG_PPROF_ADDR"))
	if addr == "" {
		return func() {}, nil
	}
	if host, _, splitErr := net.SplitHostPort(addr); splitErr != nil || (host != "127.0.0.1" && host != "localhost") {
		return nil, errors.New("HYDRO_DEBUG_PPROF_ADDR must bind to 127.0.0.1 or localhost")
	}
	// Its own mux, so the profiling endpoints exist only on this loopback
	// listener and never on http.DefaultServeMux.
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	pprofSrv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = pprofSrv.ListenAndServe() }()
	return func() { _ = pprofSrv.Close() }, nil
}

// enforceDevModeLocalBind requires --dev/HYDRO_DEV_MODE to bind only to a
// loopback address, so a development server is never reachable from outside
// the local machine.
func enforceDevModeLocalBind(address string) error {
	host, _, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("--dev/HYDRO_DEV_MODE requires a valid --address host:port, got %q: %w", address, err)
	}
	if host != "127.0.0.1" && host != "localhost" && host != "::1" {
		return fmt.Errorf("--dev/HYDRO_DEV_MODE refused: server must bind to 127.0.0.1, localhost or ::1 (got %q)", host)
	}
	return nil
}

// printDevModeWarning prints a banner so a dev-mode server is never mistaken
// for a production one.
func printDevModeWarning(stderr io.Writer) {
	fmt.Fprint(stderr, `
################################################################
#   DEV MODE - NOT FOR PRODUCTION                              #
#   Enabled via --dev/HYDRO_DEV_MODE; bound to loopback only.  #
################################################################
`)
}

type runtimeFlags struct {
	capacityProfile *string
	configPath      *string
	address         *string
	databaseDialect *string
	databaseDSN     *string
	safeMode        *bool
}

func addRuntimeFlags(flags *flag.FlagSet) runtimeFlags {
	return runtimeFlags{
		capacityProfile: flags.String("capacity-profile", "", "compact, standard or distributed"),
		configPath:      flags.String("config", "", "YAML config file"),
		address:         flags.String("address", "", "HTTP listen address"),
		databaseDialect: flags.String("database-dialect", "", "sqlite or postgres"),
		databaseDSN:     flags.String("database-dsn", "", "database connection string"),
		safeMode:        flags.Bool("safe-mode", false, "suppress external integrations and non-essential workers"),
	}
}

// loadMaintenanceConfig is loadRuntimeConfig for commands that never deliver to
// a hook or webhook, so they do not need those secrets in their environment.
func loadMaintenanceConfig(flags *flag.FlagSet, values runtimeFlags) (config.Config, error) {
	return loadConfig(flags, values, true)
}

func loadRuntimeConfig(flags *flag.FlagSet, values runtimeFlags) (config.Config, error) {
	return loadConfig(flags, values, false)
}

func loadConfig(flags *flag.FlagSet, values runtimeFlags, skipDeliverySecrets bool) (config.Config, error) {
	set := make(map[string]bool)
	flags.Visit(func(item *flag.Flag) { set[item.Name] = true })
	overrides := config.Overrides{SkipDeliverySecrets: skipDeliverySecrets}
	if set["capacity-profile"] {
		overrides.CapacityProfile = values.capacityProfile
	}
	if set["address"] {
		overrides.HTTPAddress = values.address
	}
	if set["database-dialect"] {
		overrides.DatabaseDialect = values.databaseDialect
	}
	if set["database-dsn"] {
		overrides.DatabaseDSN = values.databaseDSN
	}
	if set["safe-mode"] {
		overrides.SafeMode = values.safeMode
	}
	return config.Load(*values.configPath, os.LookupEnv, overrides)
}

func profileDatabaseOptions(budget capacity.Budget, profile capacity.Profile) database.OpenOptions {
	sqliteProfile := database.SQLiteCompactProfile
	if profile != capacity.Compact {
		sqliteProfile = database.SQLiteVPSProfile
	}
	return database.OpenOptions{
		MaxOpenConns:  budget.DBMaxOpenConns,
		MaxIdleConns:  budget.DBMaxIdleConns,
		SQLiteProfile: sqliteProfile,
	}
}

func moduleEnabled(modules []string, wanted string) bool {
	wanted = strings.ToLower(strings.TrimSpace(wanted))
	for _, module := range modules {
		if strings.ToLower(strings.TrimSpace(module)) == wanted {
			return true
		}
	}
	return false
}

func enabledModuleSet(modules []string) map[string]bool {
	result := make(map[string]bool, len(modules))
	for _, module := range modules {
		name := strings.ToLower(strings.TrimSpace(module))
		if name != "" {
			result[name] = true
		}
	}
	return result
}

func demoFrontendEnabled() bool {
	if strings.TrimSpace(os.Getenv("HYDRO_DEV_MODE")) == "1" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("HYDRO_ENABLE_DEMO_FRONTEND"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

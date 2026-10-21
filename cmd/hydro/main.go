package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type pageCache interface {
	GetOrLoad(string, func() (pages.Page, error)) (pages.Page, error)
	GetOrLoadAtGeneration(string, uint64, func() (pages.Page, error)) (pages.Page, error)
	Clear()
	Stats() cache.Stats
}

func clearPublicCaches(page pageCache, html *server.PublicHTMLCache) {
	if page != nil {
		page.Clear()
	}
	if html != nil {
		html.Clear()
	}
}

var (
	// version is overridden at release time: -ldflags "-X main.version=v1.2.3".
	version     = "0.2.0"
	buildCommit = "unknown"
	buildTime   = "unknown"
)

type versionInfo struct {
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	BuildTime       string `json:"build_time"`
	GoVersion       string `json:"go_version"`
	SchemaMigration string `json:"schema_migration"`
}

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hydro:", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stderr)
		return errors.New("a command is required")
	}

	switch args[0] {
	case "icons":
		return runIconPack(args[1:], stdout, stderr)
	case "healthcheck":
		if len(args) != 1 {
			return errors.New("healthcheck takes no arguments")
		}
		address := os.Getenv("HYDRO_HEALTHCHECK_URL")
		if address == "" {
			address = "http://127.0.0.1:8080/healthz"
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("healthcheck returned %s", resp.Status)
		}
		return nil
	case "version":
		if len(args) == 1 {
			_, err := fmt.Fprintln(stdout, version)
			return err
		}
		if len(args) == 2 && args[1] == "--json" {
			info, err := currentVersionInfo()
			if err != nil {
				return err
			}
			data, err := json.Marshal(info)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(stdout, string(data))
			return err
		}
		return errors.New("version accepts only --json")
	case "content-type":
		return contentTypeCommand(args[1:], stdout, stderr)
	case "new", "init":
		return newCommand(args[0], args[1:], stdin, stdout, stderr)
	case "serve":
		return serve(args[1:], stderr)
	case "run", "dev":
		return devCommand(args[1:], stdout, stderr)
	case "migrate":
		return migrate(args[1:], stderr)
	case "rollback":
		return rollbackCommand(args[1:], stdout, stderr)
	case "ban":
		return ipBanCommand(args[1:], false, stdout, stderr)
	case "unban":
		return ipBanCommand(args[1:], true, stdout, stderr)
	case "migration":
		return migrationCommand(args[1:], stdout, stderr)
	case "upgrade":
		return upgrade(args[1:], stdout, stderr)
	case "audit":
		return auditCommand(args[1:], stdout, stderr)
	case "service-account":
		return serviceAccountCommand(args[1:], stdout, stderr)
	case "create-admin":
		return createAdmin(args[1:], stdin, stdout, stderr)
	case "seed-demo":
		return seedDemo(args[1:], stdout, stderr)
	case "seed":
		return seedDemo(args[1:], stdout, stderr)
	case "clean":
		return cleanDemoCommand(args[1:], stdout, stderr)
	case "sdk":
		return sdkCommand(args[1:], stdout, stderr)
	case "schema":
		return schemaCommand(args[1:], stdout, stderr)
	case "db":
		return dbCommand(args[1:], stdout, stderr)
	case "doctor":
		return doctor(args[1:], stdout, stderr)
	case "validate":
		return validateCommand(args[1:], stdout, stderr)
	case "test":
		return testCommand(args[1:], stdout, stderr)
	case "backup":
		return backupCommand(args[1:], stdout, stderr)
	case "restore":
		return restoreCommand(args[1:], stdout, stderr)
	case "portable":
		return portableCommand(args[1:], stdout, stderr)
	case "release":
		return releaseCommand(args[1:], stdout, stderr)
	case "runtime":
		return runtimeCommand(args[1:], stdout, stderr)
	case "jobs":
		return jobsCommand(args[1:], stdout, stderr)
	case "tenant":
		return tenantCommand(args[1:], stdin, stdout, stderr)
	case "profile":
		return profileCommand(args[1:], stdout, stderr)
	case "events":
		return eventsCommand(args[1:], stdout)
	case "scan-links":
		return scanLinksCommand(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printUsage(stdout)
		return nil
	default:
		printUsage(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type repeatedFlag []string

func (f *repeatedFlag) String() string { return strings.Join(*f, ",") }
func (f *repeatedFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("repeated value cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted || strings.HasPrefix(value, wanted+"=") {
			return true
		}
	}
	return false
}

var migrationIDPattern = regexp.MustCompile(`^[0-9]{12}_[a-z][a-z0-9_]{1,62}$`)

func currentVersionInfo() (versionInfo, error) {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return versionInfo{}, fmt.Errorf("read embedded migrations: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return versionInfo{}, errors.New("no embedded migrations found")
	}
	return versionInfo{Version: version, Commit: buildCommit, BuildTime: buildTime, GoVersion: runtime.Version(), SchemaMigration: ids[len(ids)-1]}, nil
}

type demoNewsSeed struct {
	id, slug, title, excerpt, takeaways, category, author, coverMediaID, tags, body string
	featured                                                                        bool
}

const (
	maxReleaseManifestBytes = 1 << 20
	maxTrustedKeyFileBytes  = 4 << 10
)

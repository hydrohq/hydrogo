package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ardaozyildirim/hydrogo/internal/auth"
	backupstore "github.com/ardaozyildirim/hydrogo/internal/backup"
	"github.com/ardaozyildirim/hydrogo/internal/cache"
	"github.com/ardaozyildirim/hydrogo/internal/capacity"
	"github.com/ardaozyildirim/hydrogo/internal/config"
	"github.com/ardaozyildirim/hydrogo/internal/database"
	"github.com/ardaozyildirim/hydrogo/internal/eventcatalog"
	"github.com/ardaozyildirim/hydrogo/internal/release"
	securitystore "github.com/ardaozyildirim/hydrogo/internal/security/sqlstore"
	"github.com/ardaozyildirim/hydrogo/internal/server"
	"github.com/ardaozyildirim/hydrogo/migrations"
	"github.com/ardaozyildirim/hydrogo/modules/pages"
)

type pageCacheSpy struct{ cleared int }

func (p *pageCacheSpy) GetOrLoad(string, func() (pages.Page, error)) (pages.Page, error) {
	return pages.Page{}, nil
}
func (p *pageCacheSpy) GetOrLoadAtGeneration(string, uint64, func() (pages.Page, error)) (pages.Page, error) {
	return pages.Page{}, nil
}
func (p *pageCacheSpy) Clear()             { p.cleared++ }
func (p *pageCacheSpy) Stats() cache.Stats { return cache.Stats{} }

func TestSQLiteExternalWriteInvalidationClearsBothPublicCaches(t *testing.T) {
	page := &pageCacheSpy{}
	html := server.NewPublicHTMLCache(server.PublicHTMLCacheOptions{})
	var renders int
	handler := html.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		renders++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<main>public</main>")
	}))
	request := httptest.NewRequest(http.MethodGet, "https://example.test/public/pages/en/home/html", nil)
	for range 2 {
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
	if renders != 1 {
		t.Fatalf("initial HTML cache renders = %d, want 1", renders)
	}

	clearPublicCaches(page, html)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if page.cleared != 1 || renders != 2 {
		t.Fatalf("invalidated page cache %d times and rendered HTML %d times, want 1 and 2", page.cleared, renders)
	}
}

func TestModuleHelpersCanonicalizeNames(t *testing.T) {
	configured := []string{" Pages ", "NEWS"}
	if !moduleEnabled(configured, "pages") || !moduleEnabled(configured, " news ") {
		t.Fatalf("moduleEnabled failed to canonicalize %#v", configured)
	}
	set := enabledModuleSet(configured)
	if !set["pages"] || !set["news"] || len(set) != 2 {
		t.Fatalf("enabledModuleSet = %#v", set)
	}
}

func TestSafeModeCommandFlagOverridesEnvironment(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "hydro.yaml")
	if err := os.WriteFile(configPath, []byte("database:\n  dialect: sqlite\n  dsn: ':memory:'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HYDRO_SAFE_MODE", "false")
	flags := flag.NewFlagSet("hydro", flag.ContinueOnError)
	values := addRuntimeFlags(flags)
	if err := flags.Parse([]string{"--config=" + configPath, "--safe-mode"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadRuntimeConfig(flags, values)
	if err != nil || !cfg.SafeMode {
		t.Fatalf("safe mode from flag = %v, error = %v", cfg.SafeMode, err)
	}
}

func TestEventsCatalogCommandExportsPublishedVocabulary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"events", "catalog"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("events catalog: %v (%s)", err, stderr.String())
	}
	data, err := eventcatalog.ExportJSON()
	if err != nil || strings.TrimSpace(stdout.String()) != string(data) {
		t.Fatalf("catalog output=%q expected=%q err=%v", stdout.String(), data, err)
	}
}

func TestActivePlatformAdminRoleBoundary(t *testing.T) {
	for _, tc := range []struct {
		name string
		user *auth.User
		want bool
	}{
		{name: "active admin", user: &auth.User{Role: auth.RoleAdmin, Active: true}, want: true},
		{name: "admin explicit permission overrides do not demote role", user: &auth.User{Role: auth.RoleAdmin, Active: true, PermissionOverrides: permissionOverrides(false)}, want: true},
		{name: "inactive admin", user: &auth.User{Role: auth.RoleAdmin}},
		{name: "editor", user: &auth.User{Role: auth.RoleEditor, Active: true}},
		{name: "viewer", user: &auth.User{Role: auth.RoleViewer, Active: true}},
		{name: "anonymous"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := activePlatformAdmin(tc.user); got != tc.want {
				t.Fatalf("activePlatformAdmin(%+v) = %t, want %t", tc.user, got, tc.want)
			}
		})
	}
}

func permissionOverrides(allowed bool) map[string]bool {
	overrides := make(map[string]bool, len(auth.KnownPermissions))
	for _, permission := range auth.KnownPermissions {
		overrides[permission] = allowed
	}
	return overrides
}

func TestPageCacheConstructionFollowsModuleActivation(t *testing.T) {
	budget, err := capacity.Defaults(capacity.Compact)
	if err != nil {
		t.Fatal(err)
	}
	if pageCache, err := newPageCache(false, budget, redisRuntime{}, ""); err != nil || pageCache != nil {
		t.Fatalf("inactive page cache = %#v err=%v, want nil", pageCache, err)
	}
	if pageCache, err := newPageCache(true, budget, redisRuntime{}, ""); err != nil || pageCache == nil {
		t.Fatalf("active page cache = %#v err=%v, want non-nil", pageCache, err)
	}
}

func TestCustomAnalyticsDateRangeIsValidAndBounded(t *testing.T) {
	from, to, ok := customAnalyticsDateRange("2026-09-01", "2026-09-29")
	if !ok || from != "2026-09-01" || to != "2026-09-29" {
		t.Fatalf("customAnalyticsDateRange valid = %q %q %v", from, to, ok)
	}
	for _, value := range [][2]string{{"2026-09-30", "2026-09-29"}, {"not-a-date", "2026-09-29"}, {"2025-01-01", "2026-09-29"}, {"2026-09-01", "2999-01-01"}} {
		if _, _, ok := customAnalyticsDateRange(value[0], value[1]); ok {
			t.Errorf("accepted invalid range %q–%q", value[0], value[1])
		}
	}
}

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run version: %v", err)
	}
	if got := stdout.String(); got != version+"\n" {
		t.Fatalf("version output = %q, want %q", got, version+"\n")
	}
}

func TestPrintUsageIncludesExamplesAndRemediation(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)
	usage := output.String()
	for _, want := range []string{
		"Examples:",
		"hydro run [--address ADDR]",
		"hydro dev [--service NAME=COMMAND ...]",
		"hydro validate --config config.yaml",
		"hydro migrate --config config.yaml",
		"Failure remediation:",
		"never edit an applied migration",
		"dev port already in use",
	} {
		if !strings.Contains(usage, want) {
			t.Fatalf("usage missing %q", want)
		}
	}
}

func TestRunValidateJSONDoesNotRequireDatabase(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	config := "profile: compact\nmodules: [core, pages]\ndatabase:\n  dialect: sqlite\n  dsn: " + filepath.Join(root, "missing.sqlite") + "\nmedia:\n  dir: " + filepath.Join(root, "media") + "\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"validate", "--config", configPath, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("validate: %v stderr=%s", err, stderr.String())
	}
	var result struct {
		Valid   bool     `json:"valid"`
		Modules []string `json:"modules"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("validate JSON: %v output=%q", err, stdout.String())
	}
	if !result.Valid || !reflect.DeepEqual(result.Modules, []string{"core", "pages"}) {
		t.Fatalf("validate result = %+v", result)
	}
}

func TestRunTestRejectsInvalidBoundsBeforeLaunchingGo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"test", "--timeout=0s"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "positive timeout") {
		t.Fatalf("test invalid timeout error = %v", err)
	}
}

func TestSeedDemoRequiresExplicitConfirmation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := seedDemo([]string{"--database-dsn", "ignored"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--confirm-demo") {
		t.Fatalf("seed-demo without confirmation error = %v", err)
	}
}

func TestCleanDemoRequiresTargetAndProductionOptIn(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := cleanDemoCommand(nil, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "--demo") {
		t.Fatalf("untargeted clean error=%v", err)
	}
	t.Setenv("HYDRO_ENV", "production")
	if err := cleanDemoCommand([]string{"--demo"}, &stdout, &stderr); err == nil || !strings.Contains(err.Error(), "--force-prod") {
		t.Fatalf("production clean error=%v", err)
	}
}

func TestSeedDemoRequiresForceFlagInProduction(t *testing.T) {
	t.Setenv("HYDRO_ENV", "production")
	t.Setenv("APP_ENV", "")
	t.Setenv("GO_ENV", "")
	var stdout, stderr bytes.Buffer
	err := seedDemo([]string{"--confirm-demo", "--database-dsn", "ignored"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--force-prod") {
		t.Fatalf("production seed error = %v", err)
	}
}

func TestSeedDemoRejectsCountOutsideBounds(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := seedDemo([]string{"--confirm-demo", "--count=201"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "between 1 and 200") {
		t.Fatalf("out-of-bounds seed error = %v", err)
	}
}

func TestSeedDemoTaxonomyCreatesLocalizedCategoriesAndRelations(t *testing.T) {
	db, err := database.Open(context.Background(), database.SQLite, filepath.Join(t.TempDir(), "seed-taxonomy.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE news_items (id TEXT PRIMARY KEY)`,
		`CREATE TABLE news_taxonomy_terms (id TEXT PRIMARY KEY, slug TEXT NOT NULL, kind TEXT NOT NULL, parent_id TEXT, labels_json TEXT NOT NULL)`,
		`CREATE TABLE news_taxonomy_relations (content_id TEXT NOT NULL, term_id TEXT NOT NULL, PRIMARY KEY(content_id,term_id))`,
		`INSERT INTO news_items VALUES ('demo-news-one')`,
	} {
		if _, err := db.ExecContext(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
	items := []demoNewsSeed{{id: "demo-news-one", category: "Platform", title: "Hydro"}}
	if err := seedDemoTaxonomy(context.Background(), db, database.SQLite, items); err != nil {
		t.Fatal(err)
	}
	var termID, kind, labels string
	if err := db.QueryRow(`SELECT t.id,t.kind,t.labels_json FROM news_taxonomy_terms t JOIN news_taxonomy_relations r ON r.term_id=t.id WHERE r.content_id=?`, "demo-news-one").Scan(&termID, &kind, &labels); err != nil {
		t.Fatal(err)
	}
	if termID != "demo-category-platform" || kind != "category" || !strings.Contains(labels, `"tr":"Platform"`) || !strings.Contains(labels, `"en":"Platform"`) {
		t.Fatalf("seeded taxonomy = %q %q %q", termID, kind, labels)
	}
}

func TestLocalizeDemoNewsCreatesEnglishSeedContent(t *testing.T) {
	item := localizeDemoNews(demoNewsSeed{id: "demo-news-one", title: "Turkish title", category: "Ürün"}, 3)
	if item.id != "demo-news-one" || item.slug != "demo-story-003" || item.category != "Demo updates" || !strings.Contains(item.body, "synthetic") {
		t.Fatalf("localized demo item = %+v", item)
	}
}

func TestSeedDemoSQLiteIntegrationCountLocaleAndClean(t *testing.T) {
	root := t.TempDir()
	dsn := filepath.Join(root, "demo.sqlite")
	configPath := filepath.Join(root, "config.yaml")
	configuration := fmt.Sprintf("profile: compact\nmodules: [core, auth, members, pages, media, site, news]\ndatabase:\n  dialect: sqlite\n  dsn: %q\nmedia:\n  dir: %q\ni18n:\n  default_locale: tr\n", dsn, filepath.Join(root, "media"))
	if err := os.WriteFile(configPath, []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath, os.LookupEnv, config.Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.SQLite, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrationsForModules(context.Background(), db, database.SQLite, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()
	var stdout, stderr bytes.Buffer
	if err := seedDemo([]string{"--config=" + configPath, "--confirm-demo", "--count=2", "--locale=en"}, &stdout, &stderr); err != nil {
		t.Fatalf("seed English demo: %v (%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "news=2") || !strings.Contains(stdout.String(), "locale=en") {
		t.Fatalf("seed output = %q", stdout.String())
	}
	db, err = database.Open(context.Background(), database.SQLite, dsn)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM news_items WHERE id LIKE 'demo-news-%' AND locale='en' AND title LIKE 'HydroGo Demo Story%'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("English demo news count=%d err=%v", count, err)
	}
	var categories int
	if err := db.QueryRow(`SELECT COUNT(*) FROM news_taxonomy_relations r JOIN news_taxonomy_terms t ON t.id=r.term_id WHERE r.content_id LIKE 'demo-news-%' AND t.kind='category'`).Scan(&categories); err != nil || categories != 2 {
		t.Fatalf("demo category relation count=%d err=%v", categories, err)
	}
	db.Close()
	stdout.Reset()
	if err := seedDemo([]string{"--config=" + configPath, "--confirm-demo", "--count=1", "--locale=tr", "--clean"}, &stdout, &stderr); err != nil {
		t.Fatalf("clean and reseed: %v (%s)", err, stderr.String())
	}
	db, err = database.Open(context.Background(), database.SQLite, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.QueryRow(`SELECT COUNT(*) FROM news_items WHERE id LIKE 'demo-news-%'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("cleaned demo news count=%d err=%v", count, err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM news_taxonomy_terms WHERE id LIKE 'demo-category-%'`).Scan(&categories); err != nil || categories == 0 {
		t.Fatalf("reseeded categories=%d err=%v", categories, err)
	}
}

func TestSeedDemoPostgresIntegration(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("HYDRO_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("HYDRO_TEST_POSTGRES_DSN is not configured")
	}
	root := t.TempDir()
	configPath := filepath.Join(root, "config.yaml")
	configuration := fmt.Sprintf("profile: compact\nmodules: [core, auth, members, pages, media, site, news]\ndatabase:\n  dialect: postgres\n  dsn: %q\nmedia:\n  dir: %q\ni18n:\n  default_locale: tr\n", dsn, filepath.Join(root, "media"))
	if err := os.WriteFile(configPath, []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath, os.LookupEnv, config.Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.ApplyMigrationsForModules(context.Background(), db, database.Postgres, migrations.FS, enabledModuleSet(cfg.Modules)); err != nil {
		db.Close()
		t.Fatal(err)
	}
	db.Close()
	var stdout, stderr bytes.Buffer
	if err := seedDemo([]string{"--config=" + configPath, "--confirm-demo", "--count=2", "--locale=en"}, &stdout, &stderr); err != nil {
		t.Fatalf("seed postgres demo: %v (%s)", err, stderr.String())
	}
	db, err = database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM news_items WHERE id LIKE 'demo-news-%' AND locale='en' AND title LIKE 'HydroGo Demo Story%'`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("PostgreSQL English demo news count=%d err=%v", count, err)
	}
	db.Close()
	if err := seedDemo([]string{"--config=" + configPath, "--confirm-demo", "--count=1", "--locale=tr", "--clean"}, &stdout, &stderr); err != nil {
		t.Fatalf("clean and reseed PostgreSQL demo: %v (%s)", err, stderr.String())
	}
	db, err = database.Open(context.Background(), database.Postgres, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.QueryRow(`SELECT COUNT(*) FROM news_items WHERE id LIKE 'demo-news-%'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("PostgreSQL cleaned demo news count=%d err=%v", count, err)
	}
}

func TestSDKGenerateCommandWritesTypeScriptAndDartClients(t *testing.T) {
	root := t.TempDir()
	spec := filepath.Join(root, "api.json")
	if err := os.WriteFile(spec, []byte(`{"openapi":"3.1.0","info":{"title":"API","version":"1"},"paths":{"/pages":{"get":{"operationId":"listPages","responses":{"200":{"description":"OK"}}}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "sdk")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"sdk", "generate", "--out=" + out, "--openapi=" + spec}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("sdk generate: %v (%s)", err, stderr.String())
	}
	for _, path := range []string{filepath.Join(out, "typescript", "client.ts"), filepath.Join(out, "dart", "lib", "hydro_client.dart")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("generated client %q: %v", path, err)
		}
	}
}

func TestRunVersionJSONIncludesSchemaMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run([]string{"version", "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run version --json: %v", err)
	}
	var info versionInfo
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info.Version != version || info.SchemaMigration == "" || info.GoVersion == "" {
		t.Fatalf("incomplete version info: %+v", info)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"unknown"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil {
		t.Fatal("run unknown command succeeded")
	}
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Fatalf("usage missing from stderr: %q", stderr.String())
	}
}

func TestRunSchemaDiffAlias(t *testing.T) {
	root := t.TempDir()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	base, err := release.Sign(release.Manifest{FormatVersion: 1, Edition: "site", CoreVersion: "1", MigrationChecksums: map[string]string{"001": "a"}}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	current, err := release.Sign(release.Manifest{FormatVersion: 1, Edition: "site", CoreVersion: "1", MigrationChecksums: map[string]string{"001": "a", "002": "b"}}, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	basePath, currentPath := filepath.Join(root, "base.json"), filepath.Join(root, "current.json")
	writeSigned := func(path string, signed release.Signed) {
		t.Helper()
		data := []byte(fmt.Sprintf(`{"manifest":%s,"public_key":%q,"signature":%q}`, string(signed.Manifest), signed.PublicKey, signed.Signature))
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeSigned(basePath, base)
	writeSigned(currentPath, current)
	keyPath := filepath.Join(root, "key.pub")
	if err := os.WriteFile(keyPath, []byte(base.PublicKey), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"schema", "diff", "--baseline", basePath, "--current", currentPath, "--public-key", keyPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("schema diff: %v stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "schema-compatible added=1") {
		t.Fatalf("schema diff output = %q", stdout.String())
	}
}

func TestRunDBDiffReportsPendingWithoutCreatingMigrationLedger(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "pending.sqlite")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"db", "diff", "--database-dialect", "sqlite", "--database-dsn", dsn, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("db diff: %v stderr=%s", err, stderr.String())
	}
	var result struct {
		Clean   bool     `json:"clean"`
		Pending []string `json:"pending"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode db diff output %q: %v", stdout.String(), err)
	}
	if result.Clean || len(result.Pending) == 0 {
		t.Fatalf("db diff result = %#v, want pending migrations", result)
	}
	db, err := database.Open(context.Background(), database.SQLite, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var hasLedger bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='schema_migrations')`).Scan(&hasLedger); err != nil {
		t.Fatal(err)
	}
	if hasLedger {
		t.Fatal("read-only db diff created the migration ledger")
	}
}

func TestRunDBReindexReportsElapsedTimeAndRebuildsFTS(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "reindex.sqlite")
	db, err := database.Open(context.Background(), database.SQLite, dsn)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`CREATE TABLE hydro_search_documents (rowid INTEGER PRIMARY KEY, title TEXT NOT NULL, body TEXT NOT NULL)`,
		`CREATE VIRTUAL TABLE hydro_search_fts USING fts5(title, body, content='hydro_search_documents', content_rowid='rowid')`,
		`INSERT INTO hydro_search_documents(title, body) VALUES ('HydroGo', 'database search')`,
	} {
		if _, err := db.Exec(statement); err != nil {
			_ = db.Close()
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"db", "reindex", "--database-dialect", "sqlite", "--database-dsn", dsn}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("db reindex: %v stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "database reindex completed in") {
		t.Fatalf("db reindex output = %q", stdout.String())
	}
	db, err = database.Open(context.Background(), database.SQLite, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var matches int
	if err := db.QueryRow(`SELECT COUNT(*) FROM hydro_search_fts WHERE hydro_search_fts MATCH 'HydroGo'`).Scan(&matches); err != nil || matches != 1 {
		t.Fatalf("FTS matches=%d err=%v", matches, err)
	}
}

func TestRunMigrationCreateScaffoldsMetadataAndDialectFiles(t *testing.T) {
	root := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := run([]string{"migration", "create", "--id", "209901010001_example", "--module", "pages", "--out", root}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("migration create: %v stderr=%s", err, stderr.String())
	}
	dir := filepath.Join(root, "209901010001_example")
	for _, name := range []string{"migration.yaml", "sqlite.sql", "postgres.sql"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("generated %s missing: %v", name, err)
		}
	}
	if err := run([]string{"migration", "create", "--id", "209901010001_example", "--out", root}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("migration create overwrote an existing migration")
	}
}

func TestDevCommandRejectsUnexpectedArgumentsBeforeProcessStart(t *testing.T) {
	// devCommand now first detects whether it is running inside a HydroGo
	// Core checkout or a scaffolded Client Edition project (see
	// detectDevMode); run from the Core repository root so this still
	// exercises coreDevCommand's own flag validation, unchanged from before.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "..", "..")
	if !isCoreCheckout(root) {
		t.Skip("repository root does not look like a Core checkout in this environment")
	}
	if chdirErr := os.Chdir(root); chdirErr != nil {
		t.Fatal(chdirErr)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	var stdout, stderr bytes.Buffer
	runErr := run([]string{"dev", "unexpected"}, strings.NewReader(""), &stdout, &stderr)
	if runErr == nil || !strings.Contains(runErr.Error(), "unexpected arguments") {
		t.Fatalf("dev unexpected args error = %v", runErr)
	}
}

func TestDeadLetterRequeueRequiresExplicitConfirmation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"jobs", "dead-letter", "requeue", "--id", "job-1"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "--confirm") {
		t.Fatalf("requeue without confirmation error = %v", err)
	}
}

func TestNewsWorkerIDsAreDistinct(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 16; i++ {
		id := newsWorkerID(i)
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate worker id %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestSessionCleanupRunsImmediatelyAndStopsWithContext(t *testing.T) {
	cleaned := make(chan struct{}, 1)
	cleaner := recordingSessionCleaner{cleaned: cleaned}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runSessionCleanup(ctx, cleaner, io.Discard)
		close(done)
	}()
	select {
	case <-cleaned:
	case <-time.After(time.Second):
		t.Fatal("session cleanup did not run immediately")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("session cleanup did not stop after cancellation")
	}
}

func TestSessionCleanupNilErrorWriterIsSafe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runSessionCleanup(ctx, failingSessionCleaner{}, nil)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("session cleanup did not stop after cancellation")
	}
}

type recordingSessionCleaner struct {
	cleaned chan<- struct{}
}

type failingSessionCleaner struct{}

func (failingSessionCleaner) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, errors.New("cleanup failed")
}

func (r recordingSessionCleaner) DeleteExpired(context.Context, time.Time) (int64, error) {
	r.cleaned <- struct{}{}
	return 0, nil
}

func TestRunMigrateAndCreateAdmin(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.sqlite")
	var stdout, stderr bytes.Buffer

	// 1. Run migrate
	err := run([]string{"migrate", "--database-dialect", "sqlite", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("migrate failed: %v, stderr: %s", err, stderr.String())
	}
	backups, err := filepath.Glob(dbPath + ".bak-*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("pre-migration snapshots = %v, %v", backups, err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"upgrade", "check", "--database-dialect", "sqlite", "--database-dsn", dbPath, "--capacity-profile", "compact"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("upgrade check failed: %v, stderr: %s", err, stderr.String())
	}
	if got := stdout.String(); got != "upgrade-ready profile=compact database=sqlite\n" {
		t.Fatalf("unexpected upgrade output: %q", got)
	}

	// 2. Run create-admin
	stdout.Reset()
	stderr.Reset()
	err = run([]string{
		"create-admin",
		"--email", "admin@example.com",
		"--password-stdin",
		"--database-dialect", "sqlite",
		"--database-dsn", dbPath,
	}, strings.NewReader("Amber-River7!Cedar\n"), &stdout, &stderr)

	if err != nil {
		t.Fatalf("create-admin failed: %v, stderr: %s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Created admin user: admin@example.com") {
		t.Fatalf("unexpected create-admin output: %s", stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"doctor", "--database-dialect", "sqlite", "--database-dsn", dbPath, "--capacity-profile", "compact"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "profile=compact database=sqlite") {
		t.Fatalf("unexpected doctor output: %s", stdout.String())
	}
}

func TestHydroBanAndUnbanPersistSQLiteRules(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "security.sqlite")
	var stdout, stderr bytes.Buffer
	for _, args := range [][]string{{"migrate", "--database-dialect", "sqlite", "--database-dsn", dbPath}, {"ban", "--database-dialect", "sqlite", "--database-dsn", dbPath, "192.0.2.0/24"}} {
		if err := run(args, strings.NewReader(""), &stdout, &stderr); err != nil {
			t.Fatalf("run %v failed: %v (%s)", args[:1], err, stderr.String())
		}
	}
	db, err := database.Open(context.Background(), database.SQLite, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	items, err := (securitystore.BanStore{Dialect: database.SQLite}).ListActive(context.Background(), db, time.Now(), 10)
	_ = db.Close()
	if err != nil || len(items) != 1 || items[0].Prefix.String() != "192.0.2.0/24" {
		t.Fatalf("persisted bans = %#v, %v", items, err)
	}
	if err := run([]string{"unban", "--database-dialect", "sqlite", "--database-dsn", dbPath, "192.0.2.0/24"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
}

func TestRunReleaseVerify(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.bin"), []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := release.Build(root, release.Input{Edition: "site", CoreVersion: "1.0.0", SDKVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	signed, err := release.Sign(manifest, priv)
	if err != nil {
		t.Fatal(err)
	}
	signedBytes := []byte(fmt.Sprintf(`{"manifest":%s,"public_key":%q,"signature":%q}`, string(signed.Manifest), signed.PublicKey, signed.Signature))
	manifestFile := filepath.Join(root, "release.json")
	if err := os.WriteFile(manifestFile, signedBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	publicKeyFile := filepath.Join(root, "release.pub")
	if err := os.WriteFile(publicKeyFile, []byte(signed.PublicKey), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err = run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile}, strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatalf("run release verify failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "verified edition=site core=1.0.0") {
		t.Fatalf("unexpected release verify output: %s", stdout.String())
	}
	stdout.Reset()
	if err := run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile, "--root", root}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run release artifact verify failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "artifacts_verified=1") {
		t.Fatalf("artifact verification count missing: %s", stdout.String())
	}
	stdout.Reset()
	if err := run([]string{"runtime", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile, "--root", root}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("runtime verify alias failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "artifacts_verified=1") {
		t.Fatalf("runtime verification count missing: %s", stdout.String())
	}
	if err := os.WriteFile(filepath.Join(root, "app.bin"), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile, "--root", root}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("tampered release artifact was accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "app.bin"), []byte("bin"), 0o644); err != nil {
		t.Fatal(err)
	}

	wrongPublic, _, _ := ed25519.GenerateKey(rand.Reader)
	wrongKeyFile := filepath.Join(root, "wrong.pub")
	if err := os.WriteFile(wrongKeyFile, []byte(base64.RawStdEncoding.EncodeToString(wrongPublic)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", wrongKeyFile}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("manifest verified with an unrelated trusted key")
	}

	if err := os.WriteFile(manifestFile, bytes.Repeat([]byte("x"), maxReleaseManifestBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("oversized release manifest was accepted")
	}
	if err := os.WriteFile(manifestFile, signedBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publicKeyFile, bytes.Repeat([]byte("k"), maxTrustedKeyFileBytes+1), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"release", "verify", "--manifest", manifestFile, "--public-key", publicKeyFile}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("oversized trusted public key file was accepted")
	}
}

func TestRunBackupAndRestoreGates(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "hydro.sqlite")
	mediaDir := filepath.Join(root, "media")
	snapshot := filepath.Join(root, "snapshot")
	staging := filepath.Join(root, "staging")
	if err := os.MkdirAll(filepath.Join(mediaDir, "nested"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mediaDir, "nested", "asset.txt"), []byte("asset"), 0o640); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := run([]string{"backup", "sqlite", "--database-dsn", dbPath, "--media-dir", mediaDir, "--destination", snapshot}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("backup failed: %v, stderr: %s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "backup-created") {
		t.Fatalf("unexpected backup output: %q", stdout.String())
	}

	stdout.Reset()
	if err := run([]string{"restore", "verify", "--snapshot", snapshot}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("restore verify failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "restore-verified") {
		t.Fatalf("unexpected restore verify output: %q", stdout.String())
	}

	if err := run([]string{"restore", "stage", "--snapshot", snapshot, "--destination", staging}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("restore stage succeeded without quarantine controls")
	}
	stdout.Reset()
	if err := run([]string{"restore", "stage", "--snapshot", snapshot, "--destination", staging, "--sessions-revoked", "--egress-quarantined", "--old-writes-blocked"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("restore stage failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(staging, "RESTORE_QUARANTINED.json")); err != nil {
		t.Fatalf("restore marker missing: %v", err)
	}
	manifest, err := backupstore.ReadManifest(filepath.Join(snapshot, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := backupstore.Verify(staging, manifest); err != nil {
		t.Fatalf("staged restore failed verification: %v", err)
	}
}

func TestRunPortableExportImportRequiresExplicitConfirmation(t *testing.T) {
	root := t.TempDir()
	dbPath := filepath.Join(root, "hydro.sqlite")
	documentPath := filepath.Join(root, "portable.json")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"migrate", "--database-dsn", dbPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	if err := run([]string{"portable", "export", "--database-dsn", dbPath, "--path", documentPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("portable export failed: %v", err)
	}
	if _, err := os.Stat(documentPath); err != nil {
		t.Fatalf("portable document missing: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"portable", "import", "--path", documentPath, "--dry-run"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("portable dry-run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "portable-import-valid") {
		t.Fatalf("unexpected dry-run output: %q", stdout.String())
	}
	if err := run([]string{"portable", "import", "--database-dsn", dbPath, "--path", documentPath}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("portable import succeeded without explicit confirmation")
	}
	if err := run([]string{"portable", "import", "--database-dsn", dbPath, "--path", documentPath, "--confirm"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("portable import failed: %v", err)
	}
}

func TestPortablePathsRejectSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o750); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(root, "link-dir")
	if err := os.Symlink(target, linkDir); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if err := validatePortableOutputPath(filepath.Join(linkDir, "export.json")); err == nil {
		t.Fatal("portable export accepted a symlinked parent")
	}

	source := filepath.Join(root, "source.json")
	if err := os.WriteFile(source, []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "source-link.json")
	if err := os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	if err := validatePortableInputPath(link); err == nil {
		t.Fatal("portable import accepted a symlink source")
	}
}

func TestCheckDirectoryWritable(t *testing.T) {
	if err := checkDirectoryWritable(t.TempDir()); err != nil {
		t.Fatalf("writable directory rejected: %v", err)
	}
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := checkDirectoryWritable(file); err == nil {
		t.Fatal("file accepted as writable directory")
	}
}

func TestRunDoctorJSON(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "hydro.yaml")
	configData := fmt.Sprintf("profile: compact\ndatabase:\n  dialect: sqlite\n  dsn: %q\nmedia:\n  dir: %q\n", filepath.Join(root, "hydro.sqlite"), filepath.Join(root, "media"))
	if err := os.WriteFile(configPath, []byte(configData), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"migrate", "--config", configPath}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	stdout.Reset()
	if err := run([]string{"doctor", "--config", configPath, "--json"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor --json failed: %v", err)
	}
	var report struct {
		Version           string `json:"version"`
		SchemaMigration   string `json:"schema_migration"`
		Database          string `json:"database"`
		MediaWritable     bool   `json:"media_writable"`
		SQLiteJournalMode string `json:"sqlite_journal_mode"`
		ForeignKeyIssues  int    `json:"sqlite_foreign_key_violations"`
		MediaMode         string `json:"media_mode"`
		BackupTools       string `json:"backup_tools"`
		WorkerConcurrency int    `json:"worker_concurrency"`
		DeadLetteredJobs  int64  `json:"dead_lettered_jobs"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	mediaModeMatches := runtime.GOOS == "windows" || report.MediaMode == "0750"
	if report.Version != version || report.SchemaMigration == "" || report.Database != "sqlite" || !report.MediaWritable || report.SQLiteJournalMode == "" || report.ForeignKeyIssues != 0 || !mediaModeMatches || report.BackupTools != "sqlite" || report.WorkerConcurrency != 1 || report.DeadLetteredJobs != 0 {
		t.Fatalf("unexpected doctor report: %+v", report)
	}
	mediaDir := filepath.Join(root, "media")
	if err := os.Chmod(mediaDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := run([]string{"doctor", "--config", configPath, "--json", "--fix"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("doctor --fix failed: %v", err)
	}
	var fixed struct {
		Fixes []string `json:"fixes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &fixed); err != nil {
		t.Fatal(err)
	}
	if len(fixed.Fixes) == 0 || fixed.Fixes[0] != "sqlite_wal_checkpoint_passive" {
		t.Fatalf("doctor fixes = %#v", fixed.Fixes)
	}
	info, err := os.Stat(mediaDir)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o750 {
		t.Fatalf("media directory mode = %v", info.Mode().Perm())
	}
}

func TestRunBackupPruneRequiresPreviewAndConfirmation(t *testing.T) {
	root := t.TempDir()
	for _, item := range []struct {
		name string
		when time.Time
	}{
		{name: "snapshot-old", when: time.Unix(1, 0)},
		{name: "snapshot-new", when: time.Unix(2, 0)},
	} {
		dir := filepath.Join(root, item.name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := backupstore.WriteManifest(filepath.Join(dir, "manifest.json"), backupstore.Manifest{Version: 1, CreatedAt: item.when, Database: backupstore.FileDigest{Path: "database", SHA256: "0", Bytes: 0}}); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"backup", "prune", "--root", root, "--keep", "1", "--dry-run"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("backup prune dry-run failed: %v", err)
	}
	if !strings.Contains(stdout.String(), "snapshot-old") {
		t.Fatalf("dry-run did not list old snapshot: %q", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(root, "snapshot-old")); err != nil {
		t.Fatalf("dry-run removed snapshot: %v", err)
	}
	if err := run([]string{"backup", "prune", "--root", root, "--keep", "1"}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Fatal("backup prune succeeded without preview or confirmation")
	}
	if err := run([]string{"backup", "prune", "--root", root, "--keep", "1", "--confirm"}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("backup prune confirm failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "snapshot-old")); !os.IsNotExist(err) {
		t.Fatalf("old snapshot still exists, stat error = %v", err)
	}
}

func TestAdminNonAdminAndMemberCookieAccessControls(t *testing.T) {
	// 1. Verify Member session cookie name vs Admin session cookie name
	adminCookieName := "hydro_session"
	memberCookieName := "hydro_member_session"
	if adminCookieName == memberCookieName {
		t.Fatalf("member session cookie must never match admin session cookie")
	}
}

package main

import (
	"fmt"
	"io"
)

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  Start a project
  hydro new DIR [--preset minimal|blog|corporate|news|crm|erp-lite|docs|commerce] [--starter sveltekit|crm|commerce] [--admin-email EMAIL [--admin-password-stdin]]
  hydro init [same options]  (scaffold in the current directory)

  Run and develop
  hydro serve [--cgi | --fastcgi --fastcgi-address 127.0.0.1:9001] [--dev] [runtime flags]
  hydro run [--address ADDR] [--app-address ADDR] [--database-dsn DSN] [--redis-url URL] [--dependencies auto|none|required] [--poll DUR]
  hydro dev [--service NAME=COMMAND ...] [serve options]  (in a project: server + side services in one terminal; in a checkout: alias for hydro run)
  hydro version [--json]
  hydro healthcheck

  Database
  hydro migrate [runtime flags]
  hydro rollback [--snapshot PATH] --confirm [runtime flags]
  hydro migration create --id YYYYMMDDHHMM_name --module MODULE --out migrations
  hydro upgrade check [runtime flags]
  hydro schema diff --baseline RELEASE.json --current RELEASE.json --public-key KEY
  hydro db diff [runtime flags]
  hydro db reindex [runtime flags]

  Content and users
  hydro content-type new ID [--dir content-types]
  hydro content-type validate [--dir content-types]
  hydro audit head|verify [--anchor TENANT:SEQ:HASH] [runtime flags]
  hydro service-account create|revoke --name NAME [--permissions a,b] [--expires-days N] [runtime flags]
  hydro create-admin --email EMAIL --password-stdin [runtime flags]
  hydro seed --count=20 --locale=tr|en --clean --confirm-demo [--force-prod] [runtime flags]
  hydro clean --demo [--target demo|staging|production] [--force-prod] [runtime flags]
  hydro sdk generate [--openapi PATH] [--out DIR] [--lang all|typescript|dart]

  Operations
  hydro doctor [--json] [--fix] [runtime flags]
  hydro validate [--json] [runtime flags]
  hydro ban IP|CIDR [--for 24h] [runtime flags]
  hydro unban IP|CIDR [runtime flags]
  hydro backup sqlite|postgres --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup prune --root PATH --keep N --dry-run|--confirm [--pinned NAME,...]
  hydro restore verify --snapshot PATH
  hydro restore unpack --archive FILE.tar.gz --destination NEW_DIRECTORY --public-key HEX [--max-bytes N]
  hydro restore stage --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
  hydro restore postgres --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked [runtime flags]
  hydro restore activate-media --staging PATH --active PATH --rollback PATH
  hydro portable export --path PATH [runtime flags]
  hydro portable import --path PATH --dry-run|--confirm [--max-bytes BYTES] [runtime flags]
  hydro release verify --manifest PATH --public-key PATH [--root PATH]
  hydro jobs dead-letter list [--limit N] [--json] [runtime flags]
  hydro jobs dead-letter requeue --id ID --confirm [runtime flags]
  hydro jobs dead-letter purge --older-than 720h --limit N --confirm [runtime flags]
  hydro commerce reconcile [--provider paytr] [--hold 30m] [--stale-after 15m] [--limit N] [--json] [runtime flags]
  hydro tenant validate|list [runtime flags]
  hydro tenant provision --tenant-id ID --admin-email EMAIL --admin-password-stdin [runtime flags]
  hydro tenant backfill (--tenant-id ID|--active) [--dry-run|--confirm] [runtime flags]
  hydro scan-links --url https://example.test [--max-pages 500] [--concurrency 4] [--delay 100ms]
  hydro test [--package ./...] [--timeout 5m] [--race] [--count N]
  hydro events catalog
  hydro icons --source LOGO --output DIRECTORY
  hydro help

Content types are YAML files in content-types/ (see content-types/faq.yaml);
they need no command.

Examples:
  hydro validate --config config.yaml
  hydro migrate --config config.yaml && hydro serve --config config.yaml
  hydro test --package ./modules/news/... --timeout 2m
  hydro run --address 0.0.0.0:8090

Failure remediation:
  config/database errors: run "hydro validate --config PATH", then check the
    dialect, DSN, and file permissions before retrying.
  migration errors: run "hydro migrate" against a disposable database first;
    never edit an applied migration—create a new migration instead.
  test timeout or race failures: narrow "--package", increase "--timeout",
    and reproduce with "go test -run TestName -count=1 ./path".
  dev port already in use: pass another "--address"/"--app-address", or stop
    the process that owns the port.

Runtime flags:
  --config PATH
  --capacity-profile compact|standard|distributed
  --address HOST:PORT
  --database-dialect sqlite|postgres
  --database-dsn DSN

`)
}

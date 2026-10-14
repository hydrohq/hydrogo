package main

import (
	"fmt"
	"io"
	_ "net/http/pprof"
)

func printUsage(w io.Writer) {
	fmt.Fprint(w, `HydroGo development CLI

Usage:
  hydro version
  hydro version --json
  hydro icons --source LOGO --output DIRECTORY
  hydro migrate [runtime flags]
	  hydro run [--address ADDR] [--app-address ADDR] [--database-dsn DSN] [--redis-url URL] [--dependencies auto|none|required] [--compose-file PATH] [--poll DUR]
	  hydro dev [same options]  (backward-compatible alias for hydro run)
  hydro upgrade check [runtime flags]
  hydro create-admin --email EMAIL --password-stdin [runtime flags]
  hydro seed --count=20 --locale=tr|en --clean --confirm-demo [--force-prod] [runtime flags]
  hydro clean --demo [--target demo|staging|production] [--force-prod] [runtime flags]
  hydro doctor [--json] [--fix] [runtime flags]
  hydro validate [--json] [runtime flags]
  hydro validate generator --spec SPEC --root OUTPUT
  hydro validate api --baseline OPENAPI.json --current OPENAPI.json
  hydro frontend check --manifest PATH [--json]
  hydro test [--package ./...] [--timeout 5m] [--race] [--count N]
  hydro backup sqlite --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup postgres --destination PATH [--media-dir PATH] [runtime flags]
  hydro backup prune --root PATH --keep N --dry-run|--confirm [--pinned NAME,...]
  hydro restore verify --snapshot PATH
  hydro restore unpack --archive FILE.tar.gz --destination NEW_DIRECTORY --public-key HEX [--max-bytes N]
  hydro restore stage --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked
  hydro restore postgres --snapshot PATH --destination PATH --sessions-revoked --egress-quarantined --old-writes-blocked [runtime flags]
  hydro restore activate-media --staging PATH --active PATH --rollback PATH
  hydro portable export --path PATH [runtime flags]
  hydro portable import --path PATH --dry-run [--max-bytes BYTES] [runtime flags]
  hydro portable import --path PATH --confirm [--max-bytes BYTES] [runtime flags]
  hydro release verify --manifest PATH --public-key PATH [--root PATH]
  hydro runtime verify --manifest PATH --public-key PATH [--root PATH]
  hydro runtime cache list --root PATH
  hydro runtime cache prune --root PATH --keep VERSION,... --dry-run|--confirm
  hydro-issuer issue --customer NAME --project-id ID --private-key PATH [--edition ED] [--features F1,F2] [--expires-in DUR] [--grace-days N] [--out PATH]
  hydro license verify --license PATH [--public-key PATH] [--project-id ID] [--edition ED] [--feature F1]
  hydro license status --license PATH [--public-key PATH] [--project-id ID] [--json]
  hydro-issuer keygen [--public-out PATH] [--private-out PATH]
  hydro new PROJECT [--preset=news] [--database=sqlite] [--profile=compact] [--git]
  hydro new resource [NAME] [--fields name:type[:required][:unique],...] [--out DIR]
  hydro project new PROJECT [--preset=news] [--database=sqlite] [--profile=compact] [--with-collections] [--git]
  hydro project check [--config config.yaml] [--lock hydrogo.lock] [--json]
  hydro check [--config config.yaml] [--lock hydrogo.lock] [--json]
  hydro tenant validate [runtime flags]
  hydro tenant provision --tenant-id ID --admin-email EMAIL --admin-password-stdin [runtime flags]
  hydro tenant backfill (--tenant-id ID|--active) [--dry-run|--confirm] [runtime flags]
  hydro tenant list [runtime flags]
  hydro events catalog
  hydro fonts add FAMILY [--out public/fonts]
  hydro scan-links --url https://example.test [--max-pages 500] [--concurrency 4] [--delay 100ms]
  hydro module create --spec SPEC --out OUTPUT
  hydro module new --spec SPEC --out OUTPUT
  hydro resource create --spec SPEC --out OUTPUT
  hydro resource new --spec SPEC --out OUTPUT
  hydro resource check --root OUTPUT [--spec SPEC]
  hydro collections sync --confirm-admin-crud [--config config.yaml] [--generated generated] [--tenant ID]
  hydro api diff --baseline OPENAPI.json --current OPENAPI.json
  hydro schema diff --baseline RELEASE.json --current RELEASE.json --public-key KEY
  hydro permissions check --spec SPEC [--json]
  hydro resource create --name NAME --field name:type[:nullable] [--field ...] [--out OUTPUT]
  hydro migration create --id YYYYMMDDHHMM_name --module MODULE --out migrations
  hydro generate module|resource|crud|page|form|api|admin --spec SPEC --out OUTPUT
  hydro jobs dead-letter list [--limit N] [--json] [runtime flags]
  hydro jobs dead-letter requeue --id ID --confirm [runtime flags]
  hydro jobs dead-letter purge --older-than 720h --limit N --confirm [runtime flags]
  hydro serve [--cgi | --fastcgi --fastcgi-address 127.0.0.1:9001] [--dev] [runtime flags]
  hydro help

Examples:
  hydro validate --config config.yaml
  hydro test --package ./modules/news/... --timeout 2m
  hydro collections sync --config config.yaml --generated generated --confirm-admin-crud
  hydro module create --spec ./module.yaml --out ./modules/catalog
  hydro resource create --name products --field title:text --field price:integer
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

#!/bin/sh
# Entrypoint for the managed-container image (deploy/paas/Dockerfile).
#
# Environment (all optional unless marked):
#   HYDRO_HTTP_PUBLIC_ORIGIN   canonical https origin. Falls back to the platform URL
#                              (FLY_APP_NAME, RENDER_EXTERNAL_URL, RAILWAY_PUBLIC_DOMAIN).
#   HYDRO_JWT_SECRET, RECOVER_SECRET
#                              production secrets; generated and kept in the data dir when unset.
#                              Set them as platform secrets when the disk is ephemeral.
#   HYDRO_DATABASE_DIALECT/DSN set both to use PostgreSQL instead of the bundled SQLite file.
#   LITESTREAM_REPLICA_URL     s3://bucket/path: continuous SQLite replication, restored on boot.
#     LITESTREAM_ACCESS_KEY_ID, LITESTREAM_SECRET_ACCESS_KEY, LITESTREAM_ENDPOINT (R2), LITESTREAM_REGION
#   HYDRO_ADMIN_EMAIL, HYDRO_ADMIN_PASSWORD
#                              first admin; created only on a fresh database, or when
#                              HYDRO_BOOTSTRAP_ADMIN=true (which resets that user's password).
#   HYDRO_AUTO_MIGRATE         default true; set false to run `hydro migrate` yourself.
#   HYDRO_PAAS_TRUSTED_PROXIES comma-separated CIDRs whose X-Forwarded-For is believed.
set -eu

DATA_DIR="${HYDRO_DATA_DIR:-/app/data}"

# A freshly mounted volume is owned by root: fix that, then drop privileges.
if [ "$(id -u)" = "0" ]; then
  mkdir -p "$DATA_DIR"
  chown -R hydro:hydro "$DATA_DIR"
  exec su-exec hydro "$0" "$@"
fi

log() { echo "hydro-paas: $*" >&2; }
die() { log "error: $*"; exit 1; }

# --- public origin ----------------------------------------------------------
if [ -z "${HYDRO_HTTP_PUBLIC_ORIGIN:-}" ]; then
  if [ -n "${FLY_APP_NAME:-}" ]; then
    HYDRO_HTTP_PUBLIC_ORIGIN="https://${FLY_APP_NAME}.fly.dev"
  elif [ -n "${RENDER_EXTERNAL_URL:-}" ]; then
    HYDRO_HTTP_PUBLIC_ORIGIN="$RENDER_EXTERNAL_URL"
  elif [ -n "${RAILWAY_PUBLIC_DOMAIN:-}" ]; then
    HYDRO_HTTP_PUBLIC_ORIGIN="https://${RAILWAY_PUBLIC_DOMAIN}"
  else
    die "set HYDRO_HTTP_PUBLIC_ORIGIN to the site's https origin"
  fi
  log "HYDRO_HTTP_PUBLIC_ORIGIN not set, using platform URL $HYDRO_HTTP_PUBLIC_ORIGIN"
fi
export HYDRO_HTTP_PUBLIC_ORIGIN

export HYDRO_ENV="${HYDRO_ENV:-production}"
export HYDRO_HTTP_ADDRESS="0.0.0.0:${PORT:-8080}"
export HYDRO_SECURE_COOKIES="${HYDRO_SECURE_COOKIES:-true}"
export HYDRO_ENABLE_HSTS="${HYDRO_ENABLE_HSTS:-true}"

# --- production secrets -----------------------------------------------------
secrets_file="$DATA_DIR/.hydro-secrets.env"
gen() { head -c 48 /dev/urandom | base64 | tr -d '\n=+/' ; }
if [ -z "${HYDRO_JWT_SECRET:-}" ] || [ -z "${RECOVER_SECRET:-}${HYDRO_RECOVER_SECRET:-}" ]; then
  if [ ! -f "$secrets_file" ]; then
    umask 077
    printf 'HYDRO_JWT_SECRET=%s\nRECOVER_SECRET=%s\n' "$(gen)" "$(gen)" > "$secrets_file"
    log "generated HYDRO_JWT_SECRET and RECOVER_SECRET in $secrets_file; on an ephemeral disk set them as platform secrets so sessions survive restarts"
  fi
  # shellcheck disable=SC1090
  . "$secrets_file"
  : "${HYDRO_JWT_SECRET:?}"
fi
export HYDRO_JWT_SECRET
[ -z "${RECOVER_SECRET:-}" ] && [ -z "${HYDRO_RECOVER_SECRET:-}" ] || export RECOVER_SECRET="${RECOVER_SECRET:-$HYDRO_RECOVER_SECRET}"

# --- config -----------------------------------------------------------------
# Fly, Render, Railway and Cloudflare all reach the container from private
# address space, so trusting private ranges for X-Forwarded-For is safe: the
# container is not directly reachable from the internet.
proxies="${HYDRO_PAAS_TRUSTED_PROXIES:-10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,100.64.0.0/10,fc00::/7,fdaa::/16}"
proxy_yaml="[\"$(printf '%s' "$proxies" | sed 's/,/", "/g')\"]"
sed "s|__TRUSTED_PROXIES__|$proxy_yaml|" /app/config.template.yaml > /tmp/hydro-config.yaml
CFG="--config /tmp/hydro-config.yaml"

# --- database ---------------------------------------------------------------
use_sqlite=true
if [ -n "${HYDRO_DATABASE_DSN:-}" ] && [ "${HYDRO_DATABASE_DIALECT:-sqlite}" != "sqlite" ]; then
  use_sqlite=false
fi
fresh_db=false

if [ "$use_sqlite" = true ]; then
  db_file="$DATA_DIR/hydro.db"
  export HYDRO_DATABASE_DIALECT=sqlite
  export HYDRO_DATABASE_DSN="file:${db_file}?_busy_timeout=5000&_journal_mode=WAL"
  mkdir -p "$DATA_DIR/media"

  if [ -n "${LITESTREAM_REPLICA_URL:-}" ]; then
    {
      echo "dbs:"
      echo "  - path: $db_file"
      echo "    replicas:"
      echo "      - url: $LITESTREAM_REPLICA_URL"
      [ -n "${LITESTREAM_ENDPOINT:-}" ] && echo "        endpoint: $LITESTREAM_ENDPOINT"
      [ -n "${LITESTREAM_REGION:-}" ] && echo "        region: $LITESTREAM_REGION"
    } > /tmp/litestream.yml
    litestream restore -if-db-not-exists -if-replica-exists -config /tmp/litestream.yml "$db_file" \
      || die "litestream restore failed; refusing to start on an empty database"
  fi
  [ -f "$db_file" ] || fresh_db=true
else
  log "using the configured ${HYDRO_DATABASE_DIALECT} database (SQLite and Litestream disabled)"
fi

# --- schema + first admin ---------------------------------------------------
if [ "${HYDRO_AUTO_MIGRATE:-true}" = true ]; then
  # shellcheck disable=SC2086
  hydro migrate $CFG || die "hydro migrate failed"
fi

if [ -n "${HYDRO_ADMIN_EMAIL:-}" ] && [ -n "${HYDRO_ADMIN_PASSWORD:-}" ]; then
  if [ "$fresh_db" = true ] || [ "${HYDRO_BOOTSTRAP_ADMIN:-false}" = true ]; then
    # shellcheck disable=SC2086
    printf '%s' "$HYDRO_ADMIN_PASSWORD" | hydro create-admin --email "$HYDRO_ADMIN_EMAIL" --password-stdin $CFG \
      || die "creating the first admin failed"
  fi
fi
unset HYDRO_ADMIN_PASSWORD

# --- run --------------------------------------------------------------------
if [ "$use_sqlite" = true ] && [ -n "${LITESTREAM_REPLICA_URL:-}" ]; then
  exec litestream replicate -config /tmp/litestream.yml -exec "$*"
fi
exec "$@"

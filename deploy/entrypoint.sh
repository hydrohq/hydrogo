#!/bin/sh
set -eu

load_secret() {
  target="$1"
  file="$2"
  if [ ! -r "$file" ]; then
    echo "hydro: required secret file is unreadable: $file" >&2
    exit 1
  fi
  value=$(tr -d '\r\n' < "$file")
  if [ -z "$value" ]; then
    echo "hydro: required secret file is empty: $file" >&2
    exit 1
  fi
  export "$target=$value"
}

# Docker Compose mounts secrets as files while HydroGo's typed configuration
# intentionally reads the canonical HYDRO_* values below.
load_secret HYDRO_DATABASE_DSN /run/secrets/db_url
load_secret HYDRO_REDIS_URL /run/secrets/redis_url
load_secret HYDRO_MFA_MASTER_KEY /run/secrets/mfa_master_key
load_secret HYDRO_MEDIA_S3_ACCESS_KEY /run/secrets/s3_access_key
load_secret HYDRO_MEDIA_S3_SECRET_KEY /run/secrets/s3_secret_key
load_secret HYDRO_SMTP_PASSWORD /run/secrets/smtp_pass

export HYDRO_CAPACITY_PROFILE=standard
export HYDRO_DATABASE_DIALECT=postgres
export HYDRO_HTTP_ADDRESS=0.0.0.0:8080
export HYDRO_MEDIA_STORAGE_DRIVER=s3
export HYDRO_SECURE_COOKIES=true
export HYDRO_ENABLE_HSTS=true

exec "$@"

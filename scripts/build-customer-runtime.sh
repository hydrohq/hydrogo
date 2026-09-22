#!/usr/bin/env bash
set -euo pipefail

repo="$(pwd)"
out="${HYDRO_CUSTOMER_OUT:-${repo}/.tmp/customer-runtime}"
out="$(realpath -m -- "$out")"
repo_real="$(realpath -- "$repo")"
case "$out" in "$repo_real"|"$repo_real"/*) ;; *) echo 'HYDRO_CUSTOMER_OUT must stay inside the repository' >&2; exit 1 ;; esac
license="${HYDRO_CUSTOMER_LICENSE:-}"
trusted_keys="${HYDRO_TRUSTED_PUBLIC_KEYS:-}"
database_variant="${HYDRO_DATABASE_VARIANT:-dual}"
graphql_variant="${HYDRO_GRAPHQL:-enabled}"
otlp_variant="${HYDRO_OTLP:-enabled}"
s3_variant="${HYDRO_S3:-enabled}"
case "$database_variant" in
  dual|sqlite) ;;
  *) echo "HYDRO_DATABASE_VARIANT must be 'dual' or 'sqlite'" >&2; exit 1 ;;
esac
case "$s3_variant" in
  enabled|disabled) ;;
  *) echo "HYDRO_S3 must be 'enabled' or 'disabled'" >&2; exit 1 ;;
esac
case "$otlp_variant" in
  enabled|disabled) ;;
  *) echo "HYDRO_OTLP must be 'enabled' or 'disabled'" >&2; exit 1 ;;
esac
case "$graphql_variant" in
  enabled|disabled) ;;
  *) echo "HYDRO_GRAPHQL must be 'enabled' or 'disabled'" >&2; exit 1 ;;
esac
[[ -n "$trusted_keys" ]] || { echo 'HYDRO_TRUSTED_PUBLIC_KEYS is required' >&2; exit 1; }
[[ -n "$license" && -f "$license" ]] || { echo 'HYDRO_CUSTOMER_LICENSE must point to hydro.lic' >&2; exit 1; }
rm -rf -- "$out"
mkdir -p -- "$out"
ldflags="-s -w -X github.com/ardaozyildirim/hydrogo/internal/licensing.TrustedPublicKeysBase64=${trusted_keys}"
build_args=(-trimpath -ldflags "$ldflags")
build_tags=()
if [[ "$database_variant" == "sqlite" ]]; then
  build_tags+=(hydro_compact)
fi
if [[ "$graphql_variant" == "disabled" ]]; then
  build_tags+=(hydro_no_graphql)
fi
if [[ "$otlp_variant" == "disabled" ]]; then
  build_tags+=(hydro_no_otlp)
fi
if [[ "$s3_variant" == "disabled" ]]; then
  build_tags+=(hydro_no_s3)
fi
if ((${#build_tags[@]} > 0)); then
  tag_list=$(IFS=,; printf '%s' "${build_tags[*]}")
  build_args+=(-tags "$tag_list")
fi
if [[ "${HYDRO_OBFUSCATE:-false}" == "true" ]]; then
  command -v garble >/dev/null 2>&1 || { echo 'HYDRO_OBFUSCATE=true requires garble' >&2; exit 1; }
  garble -tiny -literals build "${build_args[@]}" -o "$out/hydro" ./cmd/hydro
else
  go build "${build_args[@]}" -o "$out/hydro" ./cmd/hydro
fi
cp -- "$license" "$out/hydro.lic"
for item in hydrogo.lock config.yaml config.example.yaml themes migrations custom frontend PROPRIETARY_NOTICE.md; do
  [[ -e "$repo/$item" ]] && cp -R -- "$repo/$item" "$out/$item"
done
if [[ -d "$out/migrations" ]]; then
  find "$out/migrations" -type f -name '*.go' -delete
fi
if find "$out" -type f \( -path '*/.git/*' -o -path '*/internal/*' -o -path '*/modules/*' -o -path '*/cmd/*' -o -name 'go.mod' -o -name 'go.sum' -o -name '*.key' -o -iname '*private*' \) -print -quit | grep -q .; then
  echo 'forbidden source/signing material found in customer package' >&2
  exit 1
fi
cat > "$out/PACKAGE-MANIFEST.md" <<EOF
# HydroGo customer runtime package

- Binary: hydro
- License: hydro.lic
- Database variant: ${database_variant}
- GraphQL: ${graphql_variant}
- OTLP: ${otlp_variant}
- S3: ${s3_variant}
- Source modules/Core: not included
- Obfuscated: ${HYDRO_OBFUSCATE:-false}
- Build commit: $(git rev-parse HEAD)
EOF
echo "PASS: customer runtime package created at $out"

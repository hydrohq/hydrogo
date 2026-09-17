#!/usr/bin/env bash
set -euo pipefail

repo="$(pwd)"
out="${HYDRO_CUSTOMER_OUT:-${repo}/.tmp/customer-runtime}"
out="$(realpath -m -- "$out")"
repo_real="$(realpath -- "$repo")"
case "$out" in "$repo_real"|"$repo_real"/*) ;; *) echo 'HYDRO_CUSTOMER_OUT must stay inside the repository' >&2; exit 1 ;; esac
license="${HYDRO_CUSTOMER_LICENSE:-}"
trusted_keys="${HYDRO_TRUSTED_PUBLIC_KEYS:-}"
[[ -n "$trusted_keys" ]] || { echo 'HYDRO_TRUSTED_PUBLIC_KEYS is required' >&2; exit 1; }
[[ -n "$license" && -f "$license" ]] || { echo 'HYDRO_CUSTOMER_LICENSE must point to hydro.lic' >&2; exit 1; }
rm -rf -- "$out"
mkdir -p -- "$out"
ldflags="-s -w -X github.com/ardaozyildirim/hydrogo/internal/licensing.TrustedPublicKeysBase64=${trusted_keys}"
if [[ "${HYDRO_OBFUSCATE:-false}" == "true" ]]; then
  command -v garble >/dev/null 2>&1 || { echo 'HYDRO_OBFUSCATE=true requires garble' >&2; exit 1; }
  garble -tiny -literals build -trimpath -ldflags "$ldflags" -o "$out/hydro" ./cmd/hydro
else
  go build -trimpath -ldflags "$ldflags" -o "$out/hydro" ./cmd/hydro
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
- Source modules/Core: not included
- Obfuscated: ${HYDRO_OBFUSCATE:-false}
- Build commit: $(git rev-parse HEAD)
EOF
echo "PASS: customer runtime package created at $out"

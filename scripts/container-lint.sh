#!/usr/bin/env bash
set -euo pipefail

dockerfile="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/Dockerfile"
test -f "$dockerfile"
for marker in \
  'FROM golang:' 'CGO_ENABLED=0' 'go build -trimpath' \
  'adduser -S -G hydro' 'USER hydro' \
  'COPY --from=build /src/hydrogo.lock' \
  'ENTRYPOINT ["/app/hydro"]' 'CMD ["serve", "--config", "/app/config.yaml"]' \
  'HEALTHCHECK --interval=30s --timeout=3s'; do
  grep -Fq "$marker" "$dockerfile"
done
if grep -Eiq 'COPY[[:space:]].*(\.lic|\.key|secret|password)' "$dockerfile"; then
  echo 'Dockerfile must not bake credentials or license keys into the image' >&2
  exit 1
fi
echo 'PASS: container image definition is non-root, credential-free, and reproducible'

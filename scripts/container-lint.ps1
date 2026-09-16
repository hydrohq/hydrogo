$ErrorActionPreference = 'Stop'

$dockerfile = Join-Path $PSScriptRoot '..\Dockerfile'
$dockerignore = Join-Path $PSScriptRoot '..\.dockerignore'
if (-not (Test-Path -LiteralPath $dockerfile)) { throw 'Dockerfile missing' }
if (-not (Test-Path -LiteralPath $dockerignore)) { throw '.dockerignore missing' }

$content = Get-Content -LiteralPath $dockerfile -Raw
$required = @(
  'FROM golang:',
  'CGO_ENABLED=0',
  'go build -trimpath',
  'adduser -S -G hydro',
  'USER hydro',
  'COPY --from=build /src/hydrogo.lock',
  'ENTRYPOINT ["/app/hydro"]',
  'CMD ["serve", "--config", "/app/config.yaml"]'
)
foreach ($needle in $required) {
  if ($content.IndexOf($needle, [StringComparison]::Ordinal) -lt 0) {
    throw "Dockerfile missing required hardening marker: $needle"
  }
}
if ($content -match '(?i)COPY\s+.*(\.lic|\.key|secret|password)') {
  throw 'Dockerfile must not bake credentials or license keys into the image'
}
Write-Output 'PASS: container image definition is non-root, credential-free, and reproducible'

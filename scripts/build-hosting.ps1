[CmdletBinding()]
param(
    [string]$Out = "dist/hydro",
    [string]$Goos = "linux",
    [string]$Goarch = "amd64"
)

$ErrorActionPreference = "Stop"

# Shared/CGI hosting: no C toolchain on the target, so this must stay a pure
# Go, statically linked binary (both the SQLite and MySQL drivers already are).
$env:GOOS = $Goos
$env:GOARCH = $Goarch
$env:CGO_ENABLED = "0"

$outDir = Split-Path -Parent $Out
if ($outDir -and -not (Test-Path -LiteralPath $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o $Out ./cmd/hydro
if ($LASTEXITCODE -ne 0) { throw "hosting build failed" }

Write-Host "PASS: built $Out for $Goos/$Goarch (CGI/FastCGI shared hosting)."
Write-Host "Upload alongside config.yaml, themes/, and deploy/hosting/.htaccess per deploy/hosting/README.md."

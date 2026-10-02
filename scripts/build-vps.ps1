[CmdletBinding()]
param(
    [string]$Out = "dist/hydro-vps",
    [string]$Goos = "linux",
    [string]$Goarch = "amd64"
)

$ErrorActionPreference = "Stop"

# VPS/container: runs as a persistent process (hydro serve / --fastcgi) under
# your own process manager, not spawned per-request by a shared-hosting CGI
# handler. Same pure-Go, CGO-free build as hosting (no cgo driver in use
# either way) so the only real difference is the output name/workflow.
$env:GOOS = $Goos
$env:GOARCH = $Goarch
$env:CGO_ENABLED = "0"

$outDir = Split-Path -Parent $Out
if ($outDir -and -not (Test-Path -LiteralPath $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o $Out ./cmd/hydro
if ($LASTEXITCODE -ne 0) { throw "VPS build failed" }

Write-Host "PASS: built $Out for $Goos/$Goarch (VPS/container, long-running hydro serve)."
Write-Host "Run behind a reverse proxy: hydro serve --config config.yaml (or --fastcgi for a hosting-provider FastCGI integration)."

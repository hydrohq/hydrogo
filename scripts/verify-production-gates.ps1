$ErrorActionPreference = "Stop"

$gateDir = Join-Path (Get-Location) ".tmp\gates"
New-Item -ItemType Directory -Force $gateDir | Out-Null
$reportPath = if ([string]::IsNullOrWhiteSpace($env:HYDRO_GATE_REPORT)) {
    Join-Path $gateDir ("production-gates-{0}.md" -f (Get-Date -Format "yyyyMMdd-HHmmss"))
} else { $env:HYDRO_GATE_REPORT }
$commit = (git rev-parse HEAD).Trim()
$profile = if ([string]::IsNullOrWhiteSpace($env:HYDRO_CAPACITY_PROFILE)) { "compact (default)" } else { $env:HYDRO_CAPACITY_PROFILE }
$goVersion = (go version).Trim()
$os = [System.Environment]::OSVersion.VersionString
@(
    "# HydroGo production gate report",
    "",
    "- Started: $(Get-Date -Format o)",
    "- Commit: $commit",
    "- Profile: $profile",
    "- Toolchain: $goVersion",
    "- OS: $os",
    "- PostgreSQL contract target: configured (credentials omitted)",
    "",
    "## Automated gates"
) | Set-Content -LiteralPath $reportPath -Encoding utf8

function Add-ReportLine {
    param([string]$Line)
    Add-Content -LiteralPath $reportPath -Value $Line -Encoding utf8
}

function Invoke-Required {
    param([string]$Name, [scriptblock]$Action)
    Write-Host "==> $Name"
    try {
        & $Action
        if ($LASTEXITCODE -ne 0) {
            throw "exit code $LASTEXITCODE"
        }
        Add-ReportLine "- PASS: $Name"
    } catch {
        Add-ReportLine "- FAIL: $Name — $($_.Exception.Message)"
        throw "$Name failed: $($_.Exception.Message)"
    }
}

Write-Host "HydroGo production gate preflight"
Write-Host "This runner executes reproducible local/CI gates; it does not replace a real restore, proxy, key-ceremony or customer-capacity drill."

if ([string]::IsNullOrWhiteSpace($env:HYDRO_TEST_POSTGRES_URL)) {
    throw "HYDRO_TEST_POSTGRES_URL is required; PostgreSQL contract tests must not be skipped."
}

Invoke-Required "gofmt" { gofmt -l . | ForEach-Object { throw "gofmt required: $_" } }
Invoke-Required "module integrity" { go mod verify }
Invoke-Required "workflow action pins" { go run ./tools/actionguard }
Invoke-Required "dependency vulnerability scan" { go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./... }
Invoke-Required "go vet" { go vet ./... }
Invoke-Required "full test suite" { go test ./... -count=1 -p 1 -timeout=5m }
Invoke-Required "shuffled test suite" { go test ./... -shuffle=on -count=1 -p 1 -timeout=5m }

$previousCgo = $env:CGO_ENABLED
$env:CGO_ENABLED = "1"
try {
    Invoke-Required "race-sensitive packages" { go test -race -p 1 ./internal/... ./modules/... ./cmd/hydro ./tests/contract -count=1 -timeout=5m }
}
finally {
    if ($null -eq $previousCgo) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $previousCgo }
}

$buildLDFLAGS = "-X main.buildCommit=production-gate -X main.buildTime=production-gate"
Invoke-Required "reproducible hydro build" {
    go build -trimpath -buildvcs=false -ldflags "$buildLDFLAGS -buildid=" -o .tmp\gates\hydro-a.exe ./cmd/hydro
    go build -trimpath -buildvcs=false -ldflags "$buildLDFLAGS -buildid=" -o .tmp\gates\hydro-b.exe ./cmd/hydro
    $a = (Get-FileHash .tmp\gates\hydro-a.exe -Algorithm SHA256).Hash
    $b = (Get-FileHash .tmp\gates\hydro-b.exe -Algorithm SHA256).Hash
    if ($a -ne $b) { throw "reproducible build digests differ: $a / $b" }
}

Invoke-Required "portable benchmark" { go test ./internal/portable -run '^$' -bench '^BenchmarkSQLitePortableRoundTrip256$' -benchtime=10x }

Write-Host "PASS: automated gates completed."
Add-ReportLine ""
Add-ReportLine "## External drills still required"
Add-ReportLine "- Real PostgreSQL restore into an empty target"
Add-ReportLine "- Reverse-proxy two-process blue/green restart and rollback"
Add-ReportLine "- External backup and signing-key recovery"
Add-ReportLine "- Production key ceremony and provenance publication"
Add-ReportLine "- Representative customer capacity, RPO and RTO measurements"
Add-ReportLine ""
Add-ReportLine "Report completed: $(Get-Date -Format o)"
Write-Host "Evidence report: $reportPath"
Write-Host "REQUIRED EXTERNAL DRILLS: real PostgreSQL restore, reverse-proxy/process rollback, external backup/key recovery, production key ceremony, and customer RPO/RTO/capacity measurement."

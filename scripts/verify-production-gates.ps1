$ErrorActionPreference = "Stop"

function Invoke-Required {
    param([string]$Name, [scriptblock]$Action)
    Write-Host "==> $Name"
    & $Action
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

Write-Host "HydroGo production gate preflight"
Write-Host "This runner executes reproducible local/CI gates; it does not replace a real restore, proxy, key-ceremony or customer-capacity drill."

if ([string]::IsNullOrWhiteSpace($env:HYDRO_TEST_POSTGRES_URL)) {
    throw "HYDRO_TEST_POSTGRES_URL is required; PostgreSQL contract tests must not be skipped."
}

Invoke-Required "gofmt" { gofmt -l . | ForEach-Object { throw "gofmt required: $_" } }
Invoke-Required "go vet" { go vet ./... }
Invoke-Required "full test suite" { go test ./... -count=1 }

$previousCgo = $env:CGO_ENABLED
$env:CGO_ENABLED = "1"
try {
    Invoke-Required "race-sensitive packages" { go test -race ./internal/cache ./internal/release ./internal/database ./internal/backup ./internal/security ./internal/server ./internal/jobs ./internal/jobs/sqlstore ./modules/news ./cmd/hydro -count=1 }
}
finally {
    if ($null -eq $previousCgo) { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $previousCgo }
}

New-Item -ItemType Directory -Force .tmp\gates | Out-Null
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
Write-Host "REQUIRED EXTERNAL DRILLS: real PostgreSQL restore, reverse-proxy/process rollback, external backup/key recovery, production key ceremony, and customer RPO/RTO/capacity measurement."

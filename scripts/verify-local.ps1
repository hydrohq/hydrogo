$ErrorActionPreference = "Stop"

function Invoke-Required {
    param([string]$Name, [scriptblock]$Action)
    Write-Host "==> $Name"
    & $Action
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

Write-Host "HydroGo local verification"

Invoke-Required "module integrity" { go mod verify }
Invoke-Required "dependency vulnerability scan" { go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./... }
Invoke-Required "gofmt" {
    $unformatted = gofmt -l . | Where-Object { $_ -notlike ".tmp*" -and $_ -notlike "*.tmp\*" }
    if ($unformatted) { throw "gofmt required: $($unformatted -join ', ')" }
}
Invoke-Required "workflow action pins" { go run ./tools/actionguard }
Invoke-Required "go vet" { go vet ./... }
Invoke-Required "full test suite" { go test ./... -count=1 -timeout=5m }
Invoke-Required "shuffled test suite" { go test ./... -shuffle=on -count=1 -timeout=5m }
$buildLDFLAGS = "-X main.buildCommit=local -X main.buildTime=local-verification"
Invoke-Required "hydro build" { go build -trimpath -buildvcs=false -ldflags $buildLDFLAGS -o .tmp\local\hydro.exe ./cmd/hydro }
Invoke-Required "client edition import boundary" { go run ./tools/importguard -root examples/client-edition }
Invoke-Required "client edition build" { Push-Location examples/client-edition; try { go build ./... } finally { Pop-Location } }
Invoke-Required "news edition import boundary" { go run ./tools/importguard -root examples/news-edition }
Invoke-Required "news edition build" { Push-Location examples/news-edition; try { go build ./... } finally { Pop-Location } }
Invoke-Required "binary health/readiness smoke" { & (Join-Path $PSScriptRoot "smoke-local.ps1") }

Write-Host "PASS: local verification completed."
Write-Host "For PostgreSQL, race and production-gate checks run scripts/verify-production-gates.ps1 with HYDRO_TEST_POSTGRES_URL set."

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
New-Item -ItemType Directory -Force .tmp\local | Out-Null
Invoke-Required "deterministic SPDX SBOM" { go run ./tools/sbom -output .tmp\local\hydro-sbom.spdx.json }
Invoke-Required "deterministic theme asset manifest" { go run ./tools/assetmanifest -root themes -output .tmp\local\asset-manifest.json }
Invoke-Required "embedded vendor asset references" { go run ./tools/assetguard -root . }
Invoke-Required "container image hardening" { & (Join-Path $PSScriptRoot "container-lint.ps1") }
Invoke-Required "Compose service binding security" { & (Join-Path $PSScriptRoot "compose-security-lint.ps1") }
Invoke-Required "tracked secret lint" { & (Join-Path $PSScriptRoot "secret-lint.ps1") }
Invoke-Required "gofmt" {
    $unformatted = gofmt -l . | Where-Object { $_ -notlike ".tmp*" -and $_ -notlike "*.tmp\*" }
    if ($unformatted) { throw "gofmt required: $($unformatted -join ', ')" }
}
Invoke-Required "workflow action pins" { go run ./tools/actionguard }
Invoke-Required "go vet" { go vet ./... }
Invoke-Required "full test suite" { go test ./... -p 1 -count=1 -timeout=5m }
Invoke-Required "risk-weighted coverage floors" { & (Join-Path $PSScriptRoot "risk-coverage.ps1") }
Invoke-Required "shuffled test suite" { go test ./... -p 1 -shuffle=on -count=1 -timeout=5m }
Invoke-Required "fuzz smoke suite" { & (Join-Path $PSScriptRoot "fuzz-smoke.ps1") }
$buildLDFLAGS = "-X main.buildCommit=local -X main.buildTime=local-verification"
Invoke-Required "hydro build" { go build -trimpath -buildvcs=false -ldflags $buildLDFLAGS -o .tmp\local\hydro.exe ./cmd/hydro }
Invoke-Required "client edition import boundary" { go run ./tools/importguard -root examples/client-edition }
Invoke-Required "client edition build" { Push-Location examples/client-edition; try { go build ./... } finally { Pop-Location } }
Invoke-Required "news edition import boundary" { go run ./tools/importguard -root examples/news-edition }
Invoke-Required "news edition build" { Push-Location examples/news-edition; try { go build ./... } finally { Pop-Location } }
Invoke-Required "generator preset smoke" { & (Join-Path $PSScriptRoot "generator-smoke.ps1") }
if (Get-Command npm -ErrorAction SilentlyContinue) {
    Invoke-Required "headless starter contract smoke" {
        Push-Location (Join-Path $PSScriptRoot "..\examples\headless-starter")
        try { npm test } finally { Pop-Location }
    }
} else {
    Write-Warning "npm bulunamadı; headless starter contract smoke atlandı (CI/browser-e2e job'ında zorunludur)."
}
Invoke-Required "theme token lint" { & (Join-Path $PSScriptRoot "theme-token-lint.ps1") }
Invoke-Required "ADR index drift check" { & (Join-Path $PSScriptRoot "adr-drift-check.ps1") }
Invoke-Required "binary health/readiness smoke" { & (Join-Path $PSScriptRoot "smoke-local.ps1") }

Write-Host "PASS: local verification completed."
Write-Host "For PostgreSQL, race and production-gate checks run scripts/verify-production-gates.ps1 with HYDRO_TEST_POSTGRES_URL set."

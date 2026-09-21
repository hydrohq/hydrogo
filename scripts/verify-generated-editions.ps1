$ErrorActionPreference = "Stop"

$repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Push-Location $repo
try {
    Write-Host "==> Project/Article generator compatibility"
    & go run ./cmd/hydro resource check --root examples/project-edition
    if ($LASTEXITCODE -ne 0) { throw "project compatibility failed" }
    & go run ./cmd/hydro resource check --root examples/article-edition
    if ($LASTEXITCODE -ne 0) { throw "article compatibility failed" }

    Write-Host "==> Generated Go migration contracts"
    & go test ./examples/project-edition/generated/project ./examples/article-edition/generated/article ./tests/contract -run "Edition|Migration" -count=1
    if ($LASTEXITCODE -ne 0) { throw "generated migration contracts failed" }

    Write-Host "==> Core migrate/serve readiness with Edition verification path"
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repo "scripts\smoke-local.ps1")
    if ($LASTEXITCODE -ne 0) { throw "core migrate/serve smoke failed" }

    Write-Host "==> Generated browser artifacts"
    Push-Location (Join-Path $repo "e2e")
    try {
        & npx playwright test custom/generated-editions.spec.ts --reporter=line
        if ($LASTEXITCODE -ne 0) { throw "generated browser contract failed" }
    } finally { Pop-Location }
    Write-Host "PASS: generated Project/Article Edition contracts"
} finally { Pop-Location }

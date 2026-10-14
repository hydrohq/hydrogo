$ErrorActionPreference = "Stop"

# Reproducible SDK/API compatibility gate. It deliberately uses a disposable
# generated project and an external TypeScript consumer; no runtime SDK
# dependency is added to HydroGo Core.
$repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$root = Join-Path $repo (".tmp\sdk-compat-" + [guid]::NewGuid().ToString("N"))
$bin = Join-Path $root "hydro.exe"
New-Item -ItemType Directory -Force $root | Out-Null
try {
    Push-Location $repo
    go test ./internal/generator ./internal/sdkdist -count=1
    go build -trimpath -buildvcs=false -o $bin .\cmd\hydro
    & $bin resource new --name sdkcompat --field title:text --out (Join-Path $root "generated")
    if ($LASTEXITCODE -ne 0) { throw "resource generation failed" }
    & $bin resource check --root (Join-Path $root "generated")
    if ($LASTEXITCODE -ne 0) { throw "generated artifact compatibility failed" }
    Push-Location (Join-Path $repo "examples\headless-starter")
    npm run check
    if ($LASTEXITCODE -ne 0) { throw "external TypeScript consumer type check failed" }
    npm test
    if ($LASTEXITCODE -ne 0) { throw "external TypeScript consumer contract failed" }
    Write-Host "PASS: HydroGo SDK/API compatibility gate"
} finally {
    Pop-Location
    if (Test-Path -LiteralPath $root) { Remove-Item -LiteralPath $root -Recurse -Force }
}

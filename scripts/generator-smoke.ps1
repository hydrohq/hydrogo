$ErrorActionPreference = "Stop"

$root = Join-Path (Get-Location) (".tmp\generator-smoke-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force $root | Out-Null
try {
    foreach ($preset in @("news", "editorial", "commerce", "compact-site", "corporate", "portal", "headless", "membership", "dashboard", "erp")) {
        $name = "smoke-$preset"
        Write-Host "==> generate $preset"
        go run ./cmd/hydro new $name --preset $preset --root $root --database sqlite --profile compact
        if ($LASTEXITCODE -ne 0) { throw "generator failed for $preset" }
        $project = Join-Path $root $name
        foreach ($file in @("config.yaml", "hydrogo.lock", "README.md", "e2e/manifest.json", "seed/data.json")) {
            if (-not (Test-Path (Join-Path $project $file))) { throw "missing $file for $preset" }
        }
        if (Test-Path (Join-Path $project "internal")) { throw "core source copied for $preset" }
    }
    Write-Host "PASS: generator smoke completed."
} finally {
    Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue
}

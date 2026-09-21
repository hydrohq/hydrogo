$ErrorActionPreference = "Stop"

$runs = if ([string]::IsNullOrWhiteSpace($env:HYDRO_TEST_STABILITY_RUNS)) { 3 } else { [int]$env:HYDRO_TEST_STABILITY_RUNS }
$output = if ([string]::IsNullOrWhiteSpace($env:HYDRO_TEST_STABILITY_REPORT)) { ".tmp\test-stability.json" } else { $env:HYDRO_TEST_STABILITY_REPORT }
if ($runs -lt 1 -or $runs -gt 20) { throw "HYDRO_TEST_STABILITY_RUNS must be an integer from 1 to 20" }

$parent = Split-Path -Parent $output
if ($parent) { New-Item -ItemType Directory -Force $parent | Out-Null }
$targets = @("./internal/auth/...", "./internal/cache/...", "./internal/database/...", "./internal/security/...", "./modules/...", "./tests/contract")
$started = (Get-Date).ToUniversalTime().ToString("o")
$passed = 0
$failed = 0

for ($run = 1; $run -le $runs; $run++) {
    Write-Host "==> stability run $run/$runs"
    & go test -shuffle=on -count=1 -p 1 -timeout=5m @targets
    if ($LASTEXITCODE -eq 0) { $passed++ } else { $failed++ }
}

$report = [ordered]@{ schema_version = 1; commit = (git rev-parse HEAD).Trim(); started_at = $started; runs = $runs; passed = $passed; failed = $failed }
$report | ConvertTo-Json | Set-Content -LiteralPath $output -Encoding utf8
if ($failed -gt 0) { throw "FAIL: $failed/$runs shuffled stability runs failed; report: $output" }
Write-Host "PASS: $passed/$runs shuffled stability runs passed; report: $output"

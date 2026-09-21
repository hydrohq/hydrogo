$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$reportDir = Join-Path $root ".tmp\gates"
New-Item -ItemType Directory -Force $reportDir | Out-Null
$report = if ([string]::IsNullOrWhiteSpace($env:HYDRO_DRILL_REPORT)) {
    Join-Path $reportDir ("local-production-drills-{0}.md" -f (Get-Date -Format "yyyyMMdd-HHmmss"))
} else { $env:HYDRO_DRILL_REPORT }

$commit = (git -C $root rev-parse HEAD).Trim()
$goVersion = (go version).Trim()
@(
    "# HydroGo local production drill report",
    "",
    "- Started: $(Get-Date -Format o)",
    "- Commit: $commit",
    "- Toolchain: $goVersion",
    "- OS: $([System.Environment]::OSVersion.VersionString)",
    "",
    "## Local drills"
) | Set-Content -LiteralPath $report -Encoding utf8

$failed = $false
function Invoke-Drill {
    param([string]$Name, [scriptblock]$Action)
    Write-Host "==> $Name"
    try {
        Push-Location $root
        try { & $Action } finally { Pop-Location }
        if ($LASTEXITCODE -ne 0) { throw "exit code $LASTEXITCODE" }
        Add-Content -LiteralPath $report -Value "- PASS: $Name" -Encoding utf8
    } catch {
        $script:failed = $true
        $message = $_.Exception.Message
        Add-Content -LiteralPath $report -Value "- FAIL: $Name - $message" -Encoding utf8
        Write-Error "$Name failed: $message"
    }
}

Invoke-Drill "isolated SQLite binary and readiness smoke" {
    & powershell -ExecutionPolicy Bypass -File (Join-Path $root "scripts\smoke-local.ps1")
}
Invoke-Drill "backup, release and portable contracts" {
    & go test ./internal/backup ./internal/release ./internal/portable -count=1 -timeout=5m
}
Invoke-Drill "blue-green cache rollback stability" {
    & go test ./internal/release -run TestBlueGreenTrafficWithConcurrentCacheInvalidation -count=20 -timeout=2m
}

Add-Content -LiteralPath $report -Value "" -Encoding utf8
Add-Content -LiteralPath $report -Value "## PostgreSQL drill" -Encoding utf8
if ($env:HYDRO_RUN_WSL_POSTGRES -eq "1") {
    Invoke-Drill "disposable WSL PostgreSQL backup and contract drill" {
        & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $root "scripts\verify-wsl-postgres-drill.ps1")
    }
} elseif ([string]::IsNullOrWhiteSpace($env:HYDRO_TEST_POSTGRES_URL)) {
    Add-Content -LiteralPath $report -Value "- PENDING: HYDRO_TEST_POSTGRES_URL is not configured" -Encoding utf8
    Write-Host "PENDING: PostgreSQL drill requires HYDRO_TEST_POSTGRES_URL."
} else {
    Invoke-Drill "PostgreSQL contract and restore tests" {
        & go test ./internal/backup ./tests/contract -count=1 -timeout=5m
    }
}

Add-Content -LiteralPath $report -Value "" -Encoding utf8
Add-Content -LiteralPath $report -Value "## Still external to this local drill" -Encoding utf8
Add-Content -LiteralPath $report -Value "- Reverse-proxy two-process deployment and rollback" -Encoding utf8
Add-Content -LiteralPath $report -Value "- External backup/key escrow recovery" -Encoding utf8
Add-Content -LiteralPath $report -Value "- Production signing-key ceremony" -Encoding utf8
Add-Content -LiteralPath $report -Value "- Customer workload capacity, RPO and RTO measurement" -Encoding utf8
Add-Content -LiteralPath $report -Value "" -Encoding utf8
Add-Content -LiteralPath $report -Value "Completed: $(Get-Date -Format o)" -Encoding utf8

Write-Host "Evidence report: $report"
if ($failed) { exit 1 }

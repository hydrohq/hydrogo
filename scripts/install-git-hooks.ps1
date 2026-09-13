$ErrorActionPreference = "Stop"

$repoRoot = git rev-parse --show-toplevel
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($repoRoot)) {
    throw "Run this script from inside the HydroGo git repository."
}
$hooksDir = Join-Path $repoRoot ".git/hooks"
New-Item -ItemType Directory -Force -Path $hooksDir | Out-Null
Copy-Item (Join-Path $repoRoot "scripts/git-hooks/pre-commit") (Join-Path $hooksDir "pre-commit") -Force
Copy-Item (Join-Path $repoRoot "scripts/git-hooks/pre-push") (Join-Path $hooksDir "pre-push") -Force
Write-Host "HydroGo git hooks installed in $hooksDir"

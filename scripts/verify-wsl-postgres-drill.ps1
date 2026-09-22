$ErrorActionPreference = "Stop"

$repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$stamp = Get-Date -Format "yyyyMMddHHmmss"
$dbName = "hydrogo_backup_drill_$stamp"
$dsn = "postgres://arda@/${dbName}?host=/var/run/postgresql"
$created = $false
try {
    & wsl psql -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE $dbName"
    if ($LASTEXITCODE -ne 0) { throw "failed to create WSL PostgreSQL database $dbName" }
    $created = $true
    $linuxRepo = "/mnt/" + $repo.Substring(0, 1).ToLowerInvariant() + $repo.Substring(2).Replace('\', '/')
    # Run package groups sequentially: both suites intentionally create/drop
    # disposable PostgreSQL schemas and concurrent `go test` processes can
    # invalidate the other suite's dump snapshot.
    $command = "cd '$linuxRepo' && export HYDRO_TEST_POSTGRES_URL='${dsn}' && go test ./internal/backup -count=1 -timeout=5m && go test ./tests/contract -count=1 -timeout=5m"
    & wsl bash -lc $command
    if ($LASTEXITCODE -ne 0) { throw "WSL PostgreSQL backup/contract drill failed with exit code $LASTEXITCODE" }
    Write-Host "PASS: WSL PostgreSQL backup/contract drill ($dbName)"
} finally {
    if ($created) {
        & wsl psql -d postgres -v ON_ERROR_STOP=1 -c "DROP DATABASE IF EXISTS $dbName" | Out-Host
    }
}

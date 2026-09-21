$ErrorActionPreference = "Stop"

$repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$port = 6393
$linuxRepo = "/mnt/" + $repo.Substring(0, 1).ToLowerInvariant() + $repo.Substring(2).Replace('\', '/')
$command = "set -e; if redis-cli -p $port ping >/dev/null 2>&1; then echo 'Redis port $port is already in use' >&2; exit 2; fi; redis-server --port $port --save '' --appendonly no --daemonize yes; trap 'redis-cli -p $port shutdown nosave >/dev/null 2>&1 || true' EXIT; cd '$linuxRepo'; HYDRO_TEST_REDIS_URL='redis://127.0.0.1:$port/15' CGO_ENABLED=1 go test -race ./internal/security -count=1"

& wsl bash -lc $command
if ($LASTEXITCODE -ne 0) {
    throw "WSL Redis/security race drill failed with exit code $LASTEXITCODE"
}
Write-Host "PASS: WSL Redis distributed admission/race drill (port $port)"

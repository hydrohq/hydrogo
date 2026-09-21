[CmdletBinding()]
param(
    [string]$OutputPath,
    [int]$WarmupRequests = 20,
    [int]$SampleCount = 5,
    [int]$SampleIntervalMs = 250
)

$ErrorActionPreference = 'Stop'
$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$runRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('hydrogo-capacity-' + [guid]::NewGuid().ToString('N'))
$binary = Join-Path $runRoot 'hydro.exe'
$profiles = @('compact', 'standard', 'distributed')
$observations = @()
$savedRedisURL = $env:HYDRO_REDIS_URL
$savedRedisPrefix = $env:HYDRO_REDIS_KEY_PREFIX

New-Item -ItemType Directory -Path $runRoot -Force | Out-Null

function Get-FreePort {
    $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $port = ([Net.IPEndPoint]$listener.LocalEndpoint).Port
    $listener.Stop()
    return $port
}

function Wait-Ready {
    param([System.Diagnostics.Process]$Process, [string]$BaseUrl)
    for ($attempt = 1; $attempt -le 40; $attempt++) {
        if ($Process.HasExited) {
            throw "capacity process exited early with code $($Process.ExitCode)"
        }
        try {
            $health = Invoke-WebRequest -Uri "$BaseUrl/healthz" -UseBasicParsing -TimeoutSec 1
            $ready = Invoke-WebRequest -Uri "$BaseUrl/readyz" -UseBasicParsing -TimeoutSec 1
            if ($health.StatusCode -eq 200 -and $ready.StatusCode -eq 200) { return }
        } catch {
            Start-Sleep -Milliseconds 250
        }
    }
    throw "capacity readiness timed out for $BaseUrl"
}

try {
    # Capacity profile smoke is intentionally local and self-contained. A
    # developer's Redis environment must not turn it into an external-service
    # availability test.
    Remove-Item Env:HYDRO_REDIS_URL -ErrorAction SilentlyContinue
    Remove-Item Env:HYDRO_REDIS_KEY_PREFIX -ErrorAction SilentlyContinue
    Write-Host '==> build capacity binary'
    & go build -trimpath -buildvcs=false -o $binary ./cmd/hydro
    if ($LASTEXITCODE -ne 0) { throw 'capacity binary build failed' }

    foreach ($profile in $profiles) {
        $profileRoot = Join-Path $runRoot $profile
        $database = Join-Path $profileRoot 'hydro.sqlite'
        $media = Join-Path $profileRoot 'media'
        $stdoutPath = Join-Path $profileRoot 'serve.stdout.log'
        $stderrPath = Join-Path $profileRoot 'serve.stderr.log'
        New-Item -ItemType Directory -Path $profileRoot, $media -Force | Out-Null

        Write-Host "==> migrate $profile profile"
        & $binary migrate --database-dialect sqlite --database-dsn $database --capacity-profile $profile
        if ($LASTEXITCODE -ne 0) { throw "migration failed for profile $profile" }

        $port = Get-FreePort
        $address = "127.0.0.1:$port"
        $process = Start-Process -FilePath $binary -ArgumentList @(
            'serve', '--database-dialect', 'sqlite', '--database-dsn', $database,
            '--address', $address, '--capacity-profile', $profile
        ) -WorkingDirectory $repositoryRoot -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -WindowStyle Hidden -PassThru
        try {
            Wait-Ready -Process $process -BaseUrl "http://$address"
            1..$WarmupRequests | ForEach-Object {
                try { Invoke-WebRequest -Uri "http://$address/healthz" -UseBasicParsing -TimeoutSec 2 | Out-Null } catch { }
            }
            $samples = @()
            for ($i = 0; $i -lt $SampleCount; $i++) {
                $process.Refresh()
                $samples += [pscustomobject]@{
                    working_set_bytes = [int64]$process.WorkingSet64
                    private_memory_bytes = [int64]$process.PrivateMemorySize64
                    handle_count = [int]$process.HandleCount
                    cpu_seconds = [double]$process.TotalProcessorTime.TotalSeconds
                }
                if ($i -lt ($SampleCount - 1)) { Start-Sleep -Milliseconds $SampleIntervalMs }
            }
            $observations += [pscustomobject]@{
                profile = $profile
                pid = $process.Id
                address = $address
                warmup_requests = $WarmupRequests
                samples = $samples
                peak_working_set_bytes = [int64](($samples | Measure-Object -Property working_set_bytes -Maximum).Maximum)
                peak_private_memory_bytes = [int64](($samples | Measure-Object -Property private_memory_bytes -Maximum).Maximum)
                peak_handle_count = [int](($samples | Measure-Object -Property handle_count -Maximum).Maximum)
            }
        }
        finally {
            if ($null -ne $process -and -not $process.HasExited) {
                Stop-Process -Id $process.Id -Force
                $process.WaitForExit()
            }
        }
    }
}
finally {
    if ($null -ne $savedRedisURL) { $env:HYDRO_REDIS_URL = $savedRedisURL }
    if ($null -ne $savedRedisPrefix) { $env:HYDRO_REDIS_KEY_PREFIX = $savedRedisPrefix }
    if (Test-Path -LiteralPath $runRoot) {
        Remove-Item -LiteralPath $runRoot -Recurse -Force
    }
}

$report = [pscustomobject]@{
    generated_at = (Get-Date).ToUniversalTime().ToString('o')
    commit = (git rev-parse HEAD).Trim()
    go_version = (go version).Trim()
    profiles = $profiles
    warmup_requests = $WarmupRequests
    sample_count = $SampleCount
    sample_interval_ms = $SampleIntervalMs
    observations = $observations
    limitations = @('SQLite single-process smoke only', 'not a customer workload or SLA', 'does not measure external PostgreSQL, Redis, proxy, media or queue capacity')
}
$json = $report | ConvertTo-Json -Depth 6
if ($OutputPath) {
    $parent = Split-Path -Parent $OutputPath
    if ($parent -and -not (Test-Path -LiteralPath $parent)) { New-Item -ItemType Directory -Path $parent -Force | Out-Null }
    Set-Content -LiteralPath $OutputPath -Value $json -Encoding UTF8
}
$json

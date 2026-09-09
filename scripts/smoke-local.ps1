$ErrorActionPreference = "Stop"

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$smokeRoot = Join-Path $repositoryRoot (Join-Path ".tmp" ("smoke-" + [guid]::NewGuid().ToString("N")))
$binary = Join-Path $smokeRoot "hydro.exe"
$database = Join-Path $smokeRoot "hydro.sqlite"
$media = Join-Path $smokeRoot "media"
$stdoutPath = Join-Path $smokeRoot "serve.stdout.log"
$stderrPath = Join-Path $smokeRoot "serve.stderr.log"
$process = $null

New-Item -ItemType Directory -Force $smokeRoot, $media | Out-Null

try {
    Write-Host "==> build smoke binary"
    & go build -trimpath -buildvcs=false -ldflags "-X main.buildCommit=smoke -X main.buildTime=smoke" -o $binary (Join-Path $repositoryRoot "cmd/hydro")
    if ($LASTEXITCODE -ne 0) { throw "smoke binary build failed" }

    Write-Host "==> migrate isolated SQLite database"
    & $binary migrate --database-dialect sqlite --database-dsn $database
    if ($LASTEXITCODE -ne 0) { throw "smoke migration failed" }

    $probe = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
    $probe.Start()
    $port = ([Net.IPEndPoint]$probe.LocalEndpoint).Port
    $probe.Stop()
    $address = "127.0.0.1:$port"

    Write-Host "==> start isolated server on $address"
    $process = Start-Process -FilePath $binary -ArgumentList @(
        "serve", "--database-dialect", "sqlite", "--database-dsn", $database, "--address", $address
    ) -WorkingDirectory $repositoryRoot -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -WindowStyle Hidden -PassThru

    $baseURL = "http://$address"
    $ready = $false
    for ($attempt = 1; $attempt -le 40; $attempt++) {
        if ($process.HasExited) {
            throw "smoke server exited early with code $($process.ExitCode)"
        }
        try {
            $health = Invoke-WebRequest -Uri "$baseURL/healthz" -UseBasicParsing -TimeoutSec 1
            $readiness = Invoke-WebRequest -Uri "$baseURL/readyz" -UseBasicParsing -TimeoutSec 1
            if ($health.StatusCode -eq 200 -and $readiness.StatusCode -eq 200 -and $readiness.Content -match "ready") {
                $ready = $true
                break
            }
        } catch {
            Start-Sleep -Milliseconds 250
        }
    }
    if (-not $ready) {
        throw "health/readiness smoke check timed out"
    }
    Write-Host "PASS: healthz and readyz returned 200 from the built binary."
}
finally {
    if ($null -ne $process -and -not $process.HasExited) {
        Stop-Process -Id $process.Id -Force
        $process.WaitForExit()
    }
    if (Test-Path -LiteralPath $stderrPath) {
        $errors = Get-Content -LiteralPath $stderrPath -Raw
        if (-not [string]::IsNullOrWhiteSpace($errors)) {
            Write-Host "smoke server stderr:"
            Write-Host $errors.Trim()
        }
    }
}

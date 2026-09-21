$ErrorActionPreference = "Stop"

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$runRoot = Join-Path $repositoryRoot (Join-Path ".tmp" ("edition-servers-" + [guid]::NewGuid().ToString("N")))
New-Item -ItemType Directory -Force $runRoot | Out-Null
$processes = @()

function Get-FreeAddress {
    $probe = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, 0)
    $probe.Start()
    $port = ([Net.IPEndPoint]$probe.LocalEndpoint).Port
    $probe.Stop()
    return "127.0.0.1:$port"
}

try {
    foreach ($edition in @("project", "article")) {
        $editionRoot = Join-Path $repositoryRoot "examples\$edition-edition"
        $binary = Join-Path $runRoot "$edition-server.exe"
        $db = Join-Path $runRoot "$edition.sqlite"
        Write-Host "==> build $edition Edition server"
        & go build -trimpath -buildvcs=false -o $binary (Join-Path $editionRoot "cmd\server")
        if ($LASTEXITCODE -ne 0) { throw "$edition Edition build failed" }
        Write-Host "==> apply operator-controlled $edition migration"
        & $binary migrate --dialect sqlite --dsn $db
        if ($LASTEXITCODE -ne 0) { throw "$edition Edition migration failed" }
        $address = Get-FreeAddress
        $stdout = Join-Path $runRoot "$edition.stdout.log"
        $stderr = Join-Path $runRoot "$edition.stderr.log"
        $process = Start-Process -FilePath $binary -ArgumentList @("serve", "--dialect", "sqlite", "--dsn", $db, "--address", $address) -WorkingDirectory $repositoryRoot -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -PassThru
        $processes += $process
        $base = "http://$address"
        $ready = $false
        for ($attempt = 1; $attempt -le 40; $attempt++) {
            if ($process.HasExited) { throw "$edition Edition server exited early with code $($process.ExitCode)" }
            try {
                $health = Invoke-WebRequest -Uri "$base/healthz" -UseBasicParsing -TimeoutSec 1
                if ($health.StatusCode -eq 200) { $ready = $true; break }
            } catch { Start-Sleep -Milliseconds 250 }
        }
        if (-not $ready) { throw "$edition Edition readiness timed out" }
        $payload = if ($edition -eq "project") { '{"id":"p-live","project_code":"P-LIVE","status":"active","title":"Live"}' } else { '{"id":"a-live","slug":"live","status":"published","title":"Live"}' }
        $path = if ($edition -eq "project") { "project" } else { "article" }
        $created = Invoke-WebRequest -Uri "$base/api/v1/$path" -Method Post -ContentType "application/json" -Body $payload -UseBasicParsing
        if ($created.StatusCode -ne 201) { throw "$edition Edition POST returned $($created.StatusCode)" }
        $id = if ($edition -eq "project") { "p-live" } else { "a-live" }
        $fetched = Invoke-WebRequest -Uri "$base/api/v1/$path/$id" -UseBasicParsing
        if ($fetched.StatusCode -ne 200) { throw "$edition Edition GET returned $($fetched.StatusCode)" }
        Write-Host "PASS: $edition Edition migration, healthz, POST and GET"
    }
}
finally {
    foreach ($process in $processes) {
        if ($null -ne $process -and -not $process.HasExited) {
            Stop-Process -Id $process.Id -Force
            $process.WaitForExit()
        }
    }
}

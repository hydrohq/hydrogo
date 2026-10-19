$ErrorActionPreference = "Stop"

if (-not $env:HYDRO_E2E_ADMIN_PASSWORD) {
    throw "Set HYDRO_E2E_ADMIN_PASSWORD to an isolated test-only password"
}

$repo = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$root = Join-Path $repo (".tmp\e2e-run-" + [guid]::NewGuid().ToString("N"))
$bin = Join-Path $root "hydro.exe"
$db = Join-Path $root "hydro.sqlite"
$media = Join-Path $root "media"
$state = Join-Path $root "auth-state.json"
$stdout = Join-Path $root "serve.stdout.log"
$stderr = Join-Path $root "serve.stderr.log"
$process = $null
$previousRedisURL = $env:HYDRO_REDIS_URL
$portProbe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
$portProbe.Start()
$port = ([System.Net.IPEndPoint]$portProbe.LocalEndpoint).Port
$portProbe.Stop()
$baseURL = "http://127.0.0.1:$port"

New-Item -ItemType Directory -Force $root, $media | Out-Null
try {
	# The browser suite proves the Compact SQLite path. Do not let a developer's
	# optional Redis/Valkey setting alter this isolated fixture.
	Remove-Item Env:HYDRO_REDIS_URL -ErrorAction SilentlyContinue
    go build -trimpath -buildvcs=false -ldflags "-X main.buildCommit=e2e -X main.buildTime=e2e" -o $bin (Join-Path $repo "cmd\hydro")
    $env:HYDRO_MEDIA_DIR = $media
    # Demo pages/products are an explicit local E2E fixture, never a production
    # default. Keep the setting scoped to this isolated child process.
    $env:HYDRO_ENABLE_DEMO_FRONTEND = "true"
    & $bin migrate --database-dialect sqlite --database-dsn $db
    & $bin seed-demo --confirm-demo --database-dialect sqlite --database-dsn $db
    if (-not $env:HYDRO_E2E_ADMIN_EMAIL) { $env:HYDRO_E2E_ADMIN_EMAIL = "e2e-admin@example.test" }
    # RedirectStandardInput avoids shell-specific native-pipe encoding and
    # keeps the password bytes identical between create-admin and setup-auth.
    $passwordFile = Join-Path $root "e2e-admin-password.txt"
    [System.IO.File]::WriteAllText($passwordFile, $env:HYDRO_E2E_ADMIN_PASSWORD + "`n", [System.Text.UTF8Encoding]::new($false))
    $createAdmin = Start-Process -FilePath $bin -ArgumentList @("create-admin", "--email", $env:HYDRO_E2E_ADMIN_EMAIL, "--password-stdin", "--database-dialect", "sqlite", "--database-dsn", $db) -RedirectStandardInput $passwordFile -NoNewWindow -Wait -PassThru
    if ($createAdmin.ExitCode -ne 0) { throw "create-admin failed with exit code $($createAdmin.ExitCode)" }
    $process = Start-Process -FilePath $bin -ArgumentList @("serve", "--database-dialect", "sqlite", "--database-dsn", $db, "--address", "127.0.0.1:$port") -WorkingDirectory $repo -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -PassThru
    $env:HYDRO_E2E_BASE_URL = $baseURL
    $env:HYDRO_E2E_STORAGE_STATE = $state
    for ($attempt = 1; $attempt -le 40; $attempt++) {
        if ($process.HasExited) { throw "E2E server exited early with code $($process.ExitCode)" }
        try {
            $health = Invoke-WebRequest -UseBasicParsing -Uri "$baseURL/healthz" -TimeoutSec 1
            if ($health.StatusCode -eq 200) { break }
        } catch { Start-Sleep -Milliseconds 250 }
    }
    Push-Location (Join-Path $repo "e2e")
    try {
        Push-Location (Join-Path $repo "packages\client")
        try {
            npm test
            if ($LASTEXITCODE -ne 0) { throw "Headless starter contract smoke failed with exit code $LASTEXITCODE" }
        } finally { Pop-Location }
        npm run setup-auth
        if ($LASTEXITCODE -ne 0) { throw "E2E auth setup failed with exit code $LASTEXITCODE" }
        npm run generate
        if ($LASTEXITCODE -ne 0) { throw "E2E generation failed with exit code $LASTEXITCODE" }
        if ($env:HYDRO_E2E_UPDATE_SNAPSHOTS -eq "1") {
            npm test -- --workers=1 --update-snapshots
        } else {
            npm test -- --workers=1
        }
        if ($LASTEXITCODE -ne 0) { throw "Playwright E2E failed with exit code $LASTEXITCODE" }
    } finally { Pop-Location }
    Write-Host "PASS: generated Admin Studio and frontend E2E completed"
} finally {
	if ($null -eq $previousRedisURL) { Remove-Item Env:HYDRO_REDIS_URL -ErrorAction SilentlyContinue } else { $env:HYDRO_REDIS_URL = $previousRedisURL }
    if ($process -and -not $process.HasExited) { Stop-Process -Id $process.Id -Force }
    if (Test-Path -LiteralPath $stderr) {
        $errors = Get-Content -LiteralPath $stderr -Raw
        if (-not [string]::IsNullOrWhiteSpace($errors)) { Write-Host $errors.Trim() }
    }
}

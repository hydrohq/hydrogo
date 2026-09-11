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

New-Item -ItemType Directory -Force $root, $media | Out-Null
try {
    go build -trimpath -buildvcs=false -ldflags "-X main.buildCommit=e2e -X main.buildTime=e2e" -o $bin (Join-Path $repo "cmd\hydro")
    $env:HYDRO_MEDIA_DIR = $media
    & $bin migrate --database-dialect sqlite --database-dsn $db
    & $bin seed-demo --database-dialect sqlite --database-dsn $db
    if (-not $env:HYDRO_E2E_ADMIN_EMAIL) { $env:HYDRO_E2E_ADMIN_EMAIL = "e2e-admin@example.test" }
    $env:HYDRO_E2E_ADMIN_PASSWORD | & $bin create-admin --email $env:HYDRO_E2E_ADMIN_EMAIL --password-stdin --database-dialect sqlite --database-dsn $db
    $process = Start-Process -FilePath $bin -ArgumentList @("serve", "--database-dialect", "sqlite", "--database-dsn", $db, "--address", "127.0.0.1:8090") -WorkingDirectory $repo -RedirectStandardOutput $stdout -RedirectStandardError $stderr -WindowStyle Hidden -PassThru
    $env:HYDRO_E2E_BASE_URL = "http://127.0.0.1:8090"
    $env:HYDRO_E2E_STORAGE_STATE = $state
    for ($attempt = 1; $attempt -le 40; $attempt++) {
        if ($process.HasExited) { throw "E2E server exited early with code $($process.ExitCode)" }
        try {
            $health = Invoke-WebRequest -UseBasicParsing -Uri "$env:HYDRO_E2E_BASE_URL/healthz" -TimeoutSec 1
            if ($health.StatusCode -eq 200) { break }
        } catch { Start-Sleep -Milliseconds 250 }
    }
    Push-Location (Join-Path $repo "e2e")
    try {
        npm run setup-auth
        npm run generate
        npm test
        if ($LASTEXITCODE -ne 0) { throw "Playwright E2E failed with exit code $LASTEXITCODE" }
    } finally { Pop-Location }
    Write-Host "PASS: generated Admin Studio and frontend E2E completed"
} finally {
    if ($process -and -not $process.HasExited) { Stop-Process -Id $process.Id -Force }
    if (Test-Path -LiteralPath $stderr) {
        $errors = Get-Content -LiteralPath $stderr -Raw
        if (-not [string]::IsNullOrWhiteSpace($errors)) { Write-Host $errors.Trim() }
    }
}

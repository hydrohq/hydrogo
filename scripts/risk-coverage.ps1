$ErrorActionPreference = "Stop"

$targets = @(
    @{ Package = "internal/auth/sqlstore"; Floor = 4.0 },
    @{ Package = "modules/members/transport"; Floor = 45.0 },
    @{ Package = "modules/api/graphql"; Floor = 65.0 }
)

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$profileRoot = Join-Path $root ".tmp\local\risk-coverage"
New-Item -ItemType Directory -Force $profileRoot | Out-Null

foreach ($target in $targets) {
    $safeName = $target.Package.Replace('/', '_')
    $profile = Join-Path $profileRoot "$safeName.out"
    Write-Host "==> risk coverage $($target.Package) (floor $($target.Floor)%)"
    Push-Location $root
    try {
        & go test "./$($target.Package)" -covermode=atomic "-coverprofile=$profile"
        if ($LASTEXITCODE -ne 0) { throw "go test failed" }
        $coverOutput = & go tool cover "-func=$profile"
        if ($LASTEXITCODE -ne 0) { throw "go tool cover failed" }
        $total = $coverOutput | Where-Object { $_ -match '^total:' } | Select-Object -First 1
        if (-not $total -or $total -notmatch '(\d+(?:\.\d+)?)%') { throw "unable to read total coverage" }
        $actual = [double]$Matches[1]
        if ($actual -lt [double]$target.Floor) {
            throw "$($target.Package) is $actual% (floor $($target.Floor)%)"
        }
        Write-Host "PASS: $($target.Package) $actual% >= $($target.Floor)%"
    } finally {
        Pop-Location
    }
}

Write-Host "PASS: risk-weighted coverage floors"

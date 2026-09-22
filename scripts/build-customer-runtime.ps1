$ErrorActionPreference = "Stop"

$repo = (Get-Location).Path
$out = if ($env:HYDRO_CUSTOMER_OUT) { $env:HYDRO_CUSTOMER_OUT } else { Join-Path $repo ".tmp/customer-runtime" }
$out = [System.IO.Path]::GetFullPath($out)
$repoPrefix = $repo.TrimEnd('\') + '\'
if ($out -eq $repo -or -not $out.StartsWith($repoPrefix, [System.StringComparison]::OrdinalIgnoreCase)) { throw "HYDRO_CUSTOMER_OUT must stay inside the repository and cannot be the repository root." }
$license = $env:HYDRO_CUSTOMER_LICENSE
$trustedKeys = $env:HYDRO_TRUSTED_PUBLIC_KEYS
$databaseVariant = if ($env:HYDRO_DATABASE_VARIANT) { $env:HYDRO_DATABASE_VARIANT.ToLowerInvariant() } else { "dual" }
if ($databaseVariant -notin @("dual", "sqlite")) { throw "HYDRO_DATABASE_VARIANT must be 'dual' or 'sqlite'." }
if ([string]::IsNullOrWhiteSpace($trustedKeys)) { throw "HYDRO_TRUSTED_PUBLIC_KEYS is required; do not build an untrusted customer runtime." }
if ([string]::IsNullOrWhiteSpace($license) -or -not (Test-Path -LiteralPath $license -PathType Leaf)) { throw "HYDRO_CUSTOMER_LICENSE must point to a signed hydro.lic file." }
$license = (Resolve-Path -LiteralPath $license).Path

if (Test-Path -LiteralPath $out) { Remove-Item -LiteralPath $out -Recurse -Force }
New-Item -ItemType Directory -Force $out | Out-Null
$binary = Join-Path $out "hydro.exe"
$ldflags = "-s -w -X github.com/ardaozyildirim/hydrogo/internal/licensing.TrustedPublicKeysBase64=$trustedKeys"
$buildArgs = @("build", "-trimpath", "-ldflags", $ldflags)
if ($databaseVariant -eq "sqlite") { $buildArgs += @("-tags", "hydro_compact") }
$buildArgs += @("-o", $binary, "./cmd/hydro")
if ($env:HYDRO_OBFUSCATE -eq "true") {
    if (-not (Get-Command garble -ErrorAction SilentlyContinue)) { throw "HYDRO_OBFUSCATE=true requires garble in the vendor build environment." }
    & garble -tiny -literals @buildArgs
} else {
    & go @buildArgs
}
if ($LASTEXITCODE -ne 0) { throw "customer runtime build failed" }

Copy-Item -LiteralPath $license -Destination (Join-Path $out "hydro.lic")
$allowlist = @("hydrogo.lock", "config.yaml", "config.example.yaml", "themes", "migrations", "custom", "frontend", "PROPRIETARY_NOTICE.md")
foreach ($item in $allowlist) {
    $source = Join-Path $repo $item
    if (Test-Path -LiteralPath $source) { Copy-Item -LiteralPath $source -Destination (Join-Path $out $item) -Recurse -Force }
}
# Migration SQL is deployment data; Go embed helpers and tests are source and
# must never be shipped in a customer package.
if (Test-Path -LiteralPath (Join-Path $out "migrations")) {
    Get-ChildItem -LiteralPath (Join-Path $out "migrations") -Recurse -File -Filter "*.go" | Remove-Item -Force
}
$forbidden = Get-ChildItem -LiteralPath $out -Recurse -Force -File | Where-Object {
    $_.FullName -match '(^|[\\/])\.git([\\/]|$)|(^|[\\/])(internal|modules|cmd)([\\/]|$)|(^|[\\/])go\.(mod|sum)$|\.key$|private'
}
if ($forbidden) { throw "forbidden source/signing material found in customer package: $($forbidden.FullName -join ', ')" }
@("# HydroGo customer runtime package", "", "- Binary: hydro.exe", "- License: hydro.lic", "- Database variant: $databaseVariant", "- Source modules/Core: not included", "- Obfuscated: $($env:HYDRO_OBFUSCATE -eq 'true')", "- Build commit: $((git rev-parse HEAD).Trim())") | Set-Content -LiteralPath (Join-Path $out "PACKAGE-MANIFEST.md") -Encoding utf8
Write-Host "PASS: customer runtime package created at $out"

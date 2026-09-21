$ErrorActionPreference = "Stop"

$required = @(
    "HYDRO_TEST_S3_ENDPOINT",
    "HYDRO_TEST_S3_ACCESS_KEY",
    "HYDRO_TEST_S3_SECRET_KEY",
    "HYDRO_TEST_S3_BUCKET"
)
$missing = @($required | Where-Object { [string]::IsNullOrWhiteSpace((Get-Item "Env:$_" -ErrorAction SilentlyContinue).Value) })
if ($missing.Count -gt 0) {
    throw "Live S3 provider drill requires: $($missing -join ', ')"
}

go test ./internal/media -run TestS3ProviderIntegration -count=1 -timeout=2m

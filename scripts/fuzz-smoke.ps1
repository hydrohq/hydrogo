$ErrorActionPreference = "Stop"

$targets = @(
    @{ Package = "./extension/v1"; Pattern = "FuzzSafeThemePath" },
    @{ Package = "./internal/config"; Pattern = "FuzzLoadEnvironmentValues" },
    @{ Package = "./internal/content"; Pattern = "FuzzValidateRichText" },
    @{ Package = "./internal/database"; Pattern = "FuzzDiscoverMigrations" },
    @{ Package = "./internal/security"; Pattern = "FuzzCSRFMiddlewareTokens" },
    @{ Package = "./internal/webhooks"; Pattern = "FuzzValidateWebhookURL" },
    @{ Package = "./internal/webhooks"; Pattern = "FuzzWebhookSignature" }
)

foreach ($target in $targets) {
    Write-Host "==> fuzz $($target.Package)/$($target.Pattern)"
    go test $target.Package -run '^$' -fuzz "^$($target.Pattern)$" -fuzztime=2s
    if ($LASTEXITCODE -ne 0) { throw "fuzz failed: $($target.Pattern)" }
}

Write-Host "PASS: fuzz smoke completed."

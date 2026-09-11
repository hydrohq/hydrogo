param(
    [string]$Output = "sbom.spdx.json"
)

$ErrorActionPreference = "Stop"

go run ./tools/sbom -output $Output
if ($LASTEXITCODE -ne 0) {
    throw "SBOM generation failed"
}

Write-Host "SBOM written to $Output"

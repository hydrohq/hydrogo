[CmdletBinding()]
param(
    [string]$Root = "docs/architecture",
    [string]$Index = "docs/architecture/README.md"
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Root -PathType Container)) { throw "ADR root does not exist: $Root" }
if (-not (Test-Path -LiteralPath $Index -PathType Leaf)) { throw "ADR index does not exist: $Index" }
$indexText = Get-Content -LiteralPath $Index -Raw
$missing = @(
    Get-ChildItem -LiteralPath $Root -File -Filter '*.md' |
        Where-Object { $_.Name -notin @('README.md', 'ADR_TEMPLATE.md') } |
        Where-Object { $indexText -notmatch [regex]::Escape($_.Name) } |
        Select-Object -ExpandProperty Name
)
if ($missing.Count -gt 0) {
    throw "ADR index drift; unlisted files:`n$($missing -join [Environment]::NewLine)"
}
Write-Host "PASS: ADR index covers $((Get-ChildItem -LiteralPath $Root -File -Filter '*.md' | Where-Object { $_.Name -notin @('README.md', 'ADR_TEMPLATE.md') }).Count) decision records."

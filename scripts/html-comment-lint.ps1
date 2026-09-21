$ErrorActionPreference = "Stop"

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$roots = @((Join-Path $repositoryRoot "themes"))
$editionRoots = Get-ChildItem -LiteralPath (Join-Path $repositoryRoot "examples") -Directory -ErrorAction SilentlyContinue |
    Where-Object { $_.Name -like "*-edition" } |
    ForEach-Object { Join-Path $_.FullName "generated" }
$roots += $editionRoots
$violations = @()
foreach ($root in $roots) {
    if (-not (Test-Path -LiteralPath $root)) { continue }
    Get-ChildItem -LiteralPath $root -Recurse -File -Filter *.html | ForEach-Object {
        $content = Get-Content -LiteralPath $_.FullName -Raw
        if ($content -match '<!--') { $violations += $_.FullName }
    }
}
if ($violations.Count -gt 0) {
    throw "HTML comments are forbidden in production-facing assets: $($violations -join ', ')"
}
Write-Host "PASS: no HTML comments in production-facing theme/generated assets"

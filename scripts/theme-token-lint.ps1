[CmdletBinding()]
param(
    [string]$Root = "themes"
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
    throw "theme root does not exist: $Root"
}

$defined = [System.Collections.Generic.HashSet[string]]::new([System.StringComparer]::OrdinalIgnoreCase)
$references = @{}
foreach ($file in Get-ChildItem -LiteralPath $Root -Recurse -File -Filter *.css) {
    $content = Get-Content -LiteralPath $file.FullName -Raw
    foreach ($match in [regex]::Matches($content, '--([a-zA-Z0-9_-]+)\s*:')) {
        [void]$defined.Add($match.Groups[1].Value)
    }
    # Only enforce references without a local fallback; var(--token, fallback)
    # is intentionally self-contained and does not require a global definition.
    foreach ($match in [regex]::Matches($content, 'var\(\s*--([a-zA-Z0-9_-]+)\s*\)')) {
        $name = $match.Groups[1].Value
        if (-not $references.ContainsKey($name)) { $references[$name] = @() }
        $references[$name] += $file.FullName
    }
}

$missing = @($references.Keys | Where-Object { -not $defined.Contains($_) } | Sort-Object)
if ($missing.Count -gt 0) {
    $details = ($missing | ForEach-Object { "--$_ (referenced by $($references[$_] -join ', '))" }) -join [Environment]::NewLine
    throw "undefined theme tokens:`n$details"
}

Write-Host "PASS: theme token lint ($($defined.Count) definitions, $($references.Count) references)."

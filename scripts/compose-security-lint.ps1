$ErrorActionPreference = "Stop"

$checks = @{
    "compose.dev.yaml" = @("127.0.0.1:5432:5432", "127.0.0.1:6379:6379")
    "deploy/local/docker-compose.storage.yml" = @("127.0.0.1:9000:9000", "127.0.0.1:9001:9001", "127.0.0.1:6379:6379")
}

foreach ($relativePath in $checks.Keys) {
    $path = Join-Path (Resolve-Path (Join-Path $PSScriptRoot "..")) $relativePath
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { throw "Compose file missing: $relativePath" }
    $content = Get-Content -LiteralPath $path -Raw
    foreach ($binding in $checks[$relativePath]) {
        if ($content -notmatch [regex]::Escape($binding)) { throw "$relativePath is missing loopback binding $binding" }
    }
    $hostBindings = [regex]::Matches($content, '(?m)^\s*-\s*"(?<host>[^":]+):\d+:\d+"\s*$')
    foreach ($match in $hostBindings) {
        if ($match.Groups['host'].Value -ne '127.0.0.1') {
            throw "$relativePath contains a non-loopback host binding: $($match.Value.Trim())"
        }
    }
    if ($content -match '(?m)^\s*-\s*"\d+:\d+"\s*$') {
        throw "$relativePath contains a host-wide local service port"
    }
    $imageLines = [regex]::Matches($content, '(?m)^\s*image:\s*(?<name>[^\s:]+):(?<tag>\S+)\s*$')
    foreach ($image in $imageLines) {
        $name = $image.Groups['name'].Value
        $tag = $image.Groups['tag'].Value
        if ($tag -eq 'latest' -or ($name -match '/redis$' -and $tag -match '^\d+(?:\.\d+)?(?:-alpine)?$') -or ($name -match '/postgres$' -and $tag -match '^\d+$')) {
            throw "$relativePath contains a mutable image tag: $($image.Value.Trim())"
        }
    }
}

Write-Host "PASS: local Compose service ports are loopback-only"

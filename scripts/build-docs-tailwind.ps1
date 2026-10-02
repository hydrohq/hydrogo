[CmdletBinding()]
param([string]$TailwindCli = "tailwindcss")

$ErrorActionPreference = 'Stop'
$cliCmd = Get-Command -Name $TailwindCli -ErrorAction SilentlyContinue
if (-not $cliCmd -and -not (Test-Path -LiteralPath $TailwindCli -PathType Leaf)) {
    throw "standalone tailwindcss CLI not found: $TailwindCli"
}

& $TailwindCli --cwd "cmd/hydro" -i "docs_portal.source.css" -o "docs_portal.css" --minify
if ($LASTEXITCODE -ne 0) {
    throw "tailwindcss build failed with exit code $LASTEXITCODE"
}
Write-Host "PASS: compiled cmd/hydro/docs_portal.css"

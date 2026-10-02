[CmdletBinding()]
param(
    [string]$TailwindCli = "tailwindcss",
    [string]$Root = "internal/ui/shadcnui"
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
    throw "shadcnui tailwind source root does not exist: $Root"
}

$cliCmd = Get-Command -Name $TailwindCli -ErrorAction SilentlyContinue
if (-not $cliCmd -and -not (Test-Path -LiteralPath $TailwindCli -PathType Leaf)) {
    throw "standalone tailwindcss CLI not found: $TailwindCli (download it from https://github.com/tailwindlabs/tailwindcss/releases and pass -TailwindCli <path>)"
}

& $TailwindCli --cwd $Root -i "input.css" -o "dist/shadcn.css" --minify
if ($LASTEXITCODE -ne 0) {
    throw "tailwindcss build failed with exit code $LASTEXITCODE"
}

Write-Host "PASS: compiled $Root/dist/shadcn.css"

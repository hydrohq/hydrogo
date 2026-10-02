[CmdletBinding()]
param(
    # Path to the standalone tailwindcss CLI executable (no Node/npm
    # involved - see docs/architecture/0338-standalone-tailwind-build-time-css.md
    # and docs/architecture/0339-tailwind-admin-visual-family.md).
    # Download the matching binary for your platform from
    # https://github.com/tailwindlabs/tailwindcss/releases (asset named
    # tailwindcss-<os>-<arch>[.exe]) and pass its path here, or place it on
    # PATH as `tailwindcss`.
    [string]$TailwindCli = "tailwindcss",
    [string]$Root = "internal/ui/aurora"
)

$ErrorActionPreference = 'Stop'
if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
    throw "aurora tailwind source root does not exist: $Root"
}

$cliCmd = Get-Command -Name $TailwindCli -ErrorAction SilentlyContinue
if (-not $cliCmd -and -not (Test-Path -LiteralPath $TailwindCli -PathType Leaf)) {
    throw "standalone tailwindcss CLI not found: $TailwindCli (download it from https://github.com/tailwindlabs/tailwindcss/releases and pass -TailwindCli <path>)"
}

& $TailwindCli --cwd $Root -i "input.css" -o "dist/aurora.css" --minify
if ($LASTEXITCODE -ne 0) {
    throw "tailwindcss build failed with exit code $LASTEXITCODE"
}

Write-Host "PASS: compiled $Root/dist/aurora.css"

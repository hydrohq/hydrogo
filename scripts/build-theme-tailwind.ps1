[CmdletBinding()]
param(
    # Path to the standalone tailwindcss CLI executable (no Node/npm
    # involved - see docs/archive/architecture/0338-standalone-tailwind-build-time-css.md
    # and docs/archive/architecture/0340-tailwind-styling-migration.md).
    # Download the matching binary for your platform from
    # https://github.com/tailwindlabs/tailwindcss/releases (asset named
    # tailwindcss-<os>-<arch>[.exe]) and pass its path here, or place it on
    # PATH as `tailwindcss`.
    [string]$TailwindCli = "tailwindcss",
    [string]$Theme = "newsroom"
)

$ErrorActionPreference = 'Stop'
$themeRoot = "themes/$Theme"
if (-not (Test-Path -LiteralPath $themeRoot -PathType Container)) {
    throw "theme root does not exist: $themeRoot"
}

$cliCmd = Get-Command -Name $TailwindCli -ErrorAction SilentlyContinue
if (-not $cliCmd -and -not (Test-Path -LiteralPath $TailwindCli -PathType Leaf)) {
    throw "standalone tailwindcss CLI not found: $TailwindCli (download it from https://github.com/tailwindlabs/tailwindcss/releases and pass -TailwindCli <path>)"
}

# themes/<name>/static/theme.source.css is the Tailwind-authored source:
# @apply rules against the theme's OWN existing selectors (ADR-0340), using
# the same "tw:" prefix convention as the admin Studio packages
# (ADR-0338/0339) - its own @import lines declare that prefix, so this
# script does not need to. Preflight is deliberately NOT imported: this is
# the theme's complete, top-level page stylesheet, and it already defines
# its own full reset (see the `*, *::before, *::after { box-sizing:
# border-box; }` rule and the rest of the top-of-file reset block) -
# importing Tailwind's preflight on top would silently change element
# defaults the theme does not expect.
& $TailwindCli --cwd $themeRoot -i "static/theme.source.css" -o "static/theme.css" --minify
if ($LASTEXITCODE -ne 0) {
    throw "tailwindcss build failed with exit code $LASTEXITCODE"
}

Write-Host "PASS: compiled $themeRoot/static/theme.css"
Write-Host "Remember to also update the theme.css?v=<hash> cache-buster in:"
Write-Host "  - themes/$Theme/layouts/base.html"
Write-Host "  - modules/members/transport/http.go"
Write-Host "  - modules/news/transport/http.go"
Write-Host "to the new file's first 8 hex chars of its SHA-256 (see 'go run ./tools/assetmanifest -root themes'), or scripts/verify-local.ps1's assetguard step will fail on mixed versions."

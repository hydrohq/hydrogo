$ErrorActionPreference = 'Stop'

$patterns = @(
  'BEGIN (RSA|EC|OPENSSH|PRIVATE) KEY',
  'AKIA[0-9A-Z]{16}',
  'gh[pousr]_[A-Za-z0-9_]{20,}',
  'sk_live_[A-Za-z0-9]{16,}',
  'rk_live_[A-Za-z0-9]{16,}',
  'xox[baprs]-[A-Za-z0-9-]{20,}'
)
$files = git ls-files -z
$bytes = [Text.Encoding]::UTF8.GetBytes($files)
$tracked = [Text.Encoding]::UTF8.GetString($bytes) -split ([char]0) | Where-Object { $_ }
$hits = @()
foreach ($file in $tracked) {
  if (-not (Test-Path -LiteralPath $file -PathType Leaf)) { continue }
  $text = Get-Content -LiteralPath $file -Raw -ErrorAction Stop
  foreach ($pattern in $patterns) {
    if ($text -match $pattern) { $hits += "$file matches $pattern" }
  }
}
if ($hits.Count -gt 0) {
  $hits | ForEach-Object { Write-Error $_ }
  exit 1
}
Write-Output 'PASS: no high-confidence credentials found in tracked files'

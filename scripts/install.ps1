# Install the hydro CLI on Windows without the Setup.exe (per-user, adds to PATH):
#   irm https://github.com/ardaozyildirim/hydrogo/releases/latest/download/install.ps1 | iex
# Set $env:HYDRO_VERSION (for example v0.2.0) to pin a version.
$ErrorActionPreference = 'Stop'
$repo = 'ardaozyildirim/hydrogo'
$version = if ($env:HYDRO_VERSION) { $env:HYDRO_VERSION } else { (Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest").tag_name }
$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
$name = "hydrogo_${version}_windows_$arch"
$base = "https://github.com/$repo/releases/download/$version"
$tmp = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid())
New-Item -ItemType Directory $tmp | Out-Null
try {
  Write-Host "Downloading HydroGo $version (windows/$arch)..."
  Invoke-WebRequest "$base/$name.zip" -OutFile "$tmp\$name.zip"
  Invoke-WebRequest "$base/SHA256SUMS" -OutFile "$tmp\SHA256SUMS"
  $expected = ((Get-Content "$tmp\SHA256SUMS" | Where-Object { $_ -match " \*?$([regex]::Escape($name)).zip$" }) -split '\s+')[0]
  $actual = (Get-FileHash "$tmp\$name.zip" -Algorithm SHA256).Hash.ToLower()
  if (-not $expected -or $expected -ne $actual) { throw 'checksum mismatch, aborting' }
  Expand-Archive "$tmp\$name.zip" -DestinationPath $tmp
  $dir = Join-Path $env:LOCALAPPDATA 'HydroGo\bin'
  New-Item -ItemType Directory $dir -Force | Out-Null
  Copy-Item "$tmp\$name\hydro.exe" "$dir\hydro.exe" -Force
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (($userPath -split ';') -notcontains $dir) {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$dir", 'User')
    Write-Host "Added $dir to your PATH. Open a new terminal to use `hydro`."
  }
  Write-Host "Installed $dir\hydro.exe"
  Write-Host 'Next: hydro new mysite --preset blog --admin-email you@example.com'
} finally {
  Remove-Item $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

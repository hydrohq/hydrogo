$ErrorActionPreference = "Stop"

$targets = @(
    @{ Package = "./internal/canonical"; Function = "FuzzPath" },
    @{ Package = "./internal/canonical"; Function = "FuzzEmail" },
    @{ Package = "./internal/portable"; Function = "FuzzUnmarshalPortable" }
)

foreach ($target in $targets) {
    Write-Host "==> Fuzz $($target.Function)"
    go test $target.Package -run '^$' -fuzz=$target.Function -fuzztime=30s -count=1
    if ($LASTEXITCODE -ne 0) {
        throw "Fuzz $($target.Function) failed with exit code $LASTEXITCODE"
    }
}

Write-Host "PASS: fuzz targets completed"

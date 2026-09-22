$ErrorActionPreference = "Stop"

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$workRoot = Join-Path $tempRoot ("hydro-edition-footprint-" + [guid]::NewGuid().ToString("N"))
$resolvedWorkRoot = [IO.Path]::GetFullPath($workRoot)
if (-not $resolvedWorkRoot.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Edition footprint output must stay inside the system temporary directory"
}

$maxBinaryBytes = 18MB
$maxDependencyPackages = 100
$forbiddenImports = @(
    "github.com/ardaozyildirim/hydrogo/cmd/hydro",
    "github.com/ardaozyildirim/hydrogo/internal/generator",
    "github.com/ardaozyildirim/hydrogo/internal/projectgen",
    "github.com/ardaozyildirim/hydrogo/internal/sdkdist",
    "github.com/ardaozyildirim/hydrogo/internal/release",
    "github.com/ardaozyildirim/hydrogo/modules/api/graphql",
    "github.com/graphql-go/graphql",
    "github.com/minio/minio-go/v7",
    "github.com/redis/go-redis/v9",
    "go.opentelemetry.io/otel/exporters/otlp",
    "google.golang.org/grpc"
)

New-Item -ItemType Directory -Path $resolvedWorkRoot | Out-Null
try {
    foreach ($edition in @("project", "article")) {
        $target = "./examples/$edition-edition/cmd/server"
        $binary = Join-Path $resolvedWorkRoot "$edition-edition.exe"
        & go build -trimpath -buildvcs=false -ldflags "-s -w -buildid=" -o $binary $target
        if ($LASTEXITCODE -ne 0) { throw "$edition Edition build failed" }

        $binaryBytes = (Get-Item -LiteralPath $binary).Length
        if ($binaryBytes -gt $maxBinaryBytes) {
            throw "$edition Edition binary is $binaryBytes bytes; budget is $maxBinaryBytes"
        }

        $imports = @(& go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' $target)
        if ($LASTEXITCODE -ne 0) { throw "$edition Edition dependency listing failed" }
        $imports = @($imports | Where-Object { $_ } | Sort-Object -Unique)
        if ($imports.Count -gt $maxDependencyPackages) {
            throw "$edition Edition has $($imports.Count) non-stdlib packages; budget is $maxDependencyPackages"
        }
        foreach ($import in $imports) {
            foreach ($forbidden in $forbiddenImports) {
                if ($import -eq $forbidden -or $import.StartsWith("$forbidden/", [StringComparison]::Ordinal)) {
                    throw "$edition Edition unexpectedly imports $import"
                }
            }
        }
        Write-Host "PASS: edition=$edition binary_bytes=$binaryBytes nonstdlib_packages=$($imports.Count)"
    }
}
finally {
    if (Test-Path -LiteralPath $resolvedWorkRoot) {
        Remove-Item -LiteralPath $resolvedWorkRoot -Recurse -Force
    }
}

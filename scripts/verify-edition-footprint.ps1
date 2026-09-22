$ErrorActionPreference = "Stop"

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
$workRoot = Join-Path $tempRoot ("hydro-edition-footprint-" + [guid]::NewGuid().ToString("N"))
$resolvedWorkRoot = [IO.Path]::GetFullPath($workRoot)
if (-not $resolvedWorkRoot.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Edition footprint output must stay inside the system temporary directory"
}

$variants = @(
    @{ Name = "dual-database"; Tags = $null; MaxBytes = 18MB; MaxPackages = 100 },
    @{ Name = "compact-sqlite"; Tags = "hydro_compact"; MaxBytes = 13MB; MaxPackages = 70 },
    @{ Name = "compact-sqlite-no-graphql"; Tags = "hydro_compact,hydro_no_graphql"; MaxBytes = 13MB; MaxPackages = 70 }
)
$baseForbiddenImports = @(
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
Push-Location $repositoryRoot
try {
    foreach ($edition in @("project", "article")) {
        $target = "./examples/$edition-edition/cmd/server"
        foreach ($variant in $variants) {
            $binary = Join-Path $resolvedWorkRoot "$edition-$($variant.Name).exe"
            $buildArgs = @("build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w -buildid=")
            $listArgs = @("list", "-deps")
            if ($variant.Tags) {
                $buildArgs += @("-tags", $variant.Tags)
                $listArgs += @("-tags", $variant.Tags)
            }
            $buildArgs += @("-o", $binary, $target)
            & go @buildArgs
            if ($LASTEXITCODE -ne 0) { throw "$edition Edition ($($variant.Name)) build failed" }

            $binaryBytes = (Get-Item -LiteralPath $binary).Length
            if ($binaryBytes -gt $variant.MaxBytes) {
                throw "$edition Edition ($($variant.Name)) binary is $binaryBytes bytes; budget is $($variant.MaxBytes)"
            }

            $listArgs += @("-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", $target)
            $imports = @(& go @listArgs)
            if ($LASTEXITCODE -ne 0) { throw "$edition Edition dependency listing failed" }
            $imports = @($imports | Where-Object { $_ } | Sort-Object -Unique)
            if ($imports.Count -gt $variant.MaxPackages) {
                throw "$edition Edition ($($variant.Name)) has $($imports.Count) non-stdlib packages; budget is $($variant.MaxPackages)"
            }

            $forbiddenImports = @($baseForbiddenImports)
            if ($variant.Tags -match "hydro_compact") {
                $forbiddenImports += @("github.com/jackc/pgx/v5", "github.com/jackc/pgpassfile", "github.com/jackc/pgservicefile", "github.com/jackc/puddle/v2")
            }
            if ($variant.Tags -match "hydro_no_graphql") {
                $forbiddenImports += @("github.com/ardaozyildirim/hydrogo/modules/api/graphql", "github.com/graphql-go/graphql")
            }
            foreach ($import in $imports) {
                foreach ($forbidden in $forbiddenImports) {
                    if ($import -eq $forbidden -or $import.StartsWith("$forbidden/", [StringComparison]::Ordinal)) {
                        throw "$edition Edition ($($variant.Name)) unexpectedly imports $import"
                    }
                }
            }
            Write-Host "PASS: edition=$edition variant=$($variant.Name) binary_bytes=$binaryBytes nonstdlib_packages=$($imports.Count)"
        }
    }

    $runtimeTags = "hydro_compact,hydro_no_graphql,hydro_no_otlp,hydro_no_s3"
    $runtimeBinary = Join-Path $resolvedWorkRoot "hydro-compact-no-graphql-no-otlp-no-s3.exe"
    & go build -tags $runtimeTags -trimpath -buildvcs=false -ldflags "-s -w -buildid=" -o $runtimeBinary ./cmd/hydro
    if ($LASTEXITCODE -ne 0) { throw "Compact SQLite/no-GraphQL Workspace build failed" }
    $runtimeBytes = (Get-Item -LiteralPath $runtimeBinary).Length
    if ($runtimeBytes -gt 29MB) { throw "Compact SQLite/no-GraphQL/no-OTLP/no-S3 Workspace binary is $runtimeBytes bytes; budget is $([int](29MB))" }
    $runtimeImports = @(& go list -deps -tags $runtimeTags -f "{{if not .Standard}}{{.ImportPath}}{{end}}" ./cmd/hydro | Where-Object { $_ } | Sort-Object -Unique)
    if ($LASTEXITCODE -ne 0) { throw "Compact SQLite/no-GraphQL/no-OTLP Workspace dependency listing failed" }
    if ($runtimeImports.Count -gt 175) { throw "Compact SQLite/no-GraphQL/no-OTLP/no-S3 Workspace has $($runtimeImports.Count) non-stdlib packages; budget is 175" }
    foreach ($import in $runtimeImports) {
        if ($import -eq "github.com/jackc/pgx/v5" -or $import.StartsWith("github.com/jackc/pgx/v5/", [StringComparison]::Ordinal) -or $import -eq "github.com/graphql-go/graphql" -or $import.StartsWith("github.com/graphql-go/graphql/", [StringComparison]::Ordinal) -or $import -eq "github.com/ardaozyildirim/hydrogo/modules/api/graphql" -or $import -like "go.opentelemetry.io/otel/exporters/otlp*" -or $import -eq "go.opentelemetry.io/proto/otlp" -or $import -like "google.golang.org/grpc*" -or $import -eq "github.com/minio/minio-go/v7" -or $import.StartsWith("github.com/minio/minio-go/v7/", [StringComparison]::Ordinal)) {
            throw "Compact SQLite/no-GraphQL/no-OTLP/no-S3 Workspace unexpectedly imports $import"
        }
    }
    Write-Host "PASS: workspace variant=compact-sqlite-no-graphql-no-otlp-no-s3 binary_bytes=$runtimeBytes nonstdlib_packages=$($runtimeImports.Count)"
}
finally {
    Pop-Location
    if (Test-Path -LiteralPath $resolvedWorkRoot) {
        Remove-Item -LiteralPath $resolvedWorkRoot -Recurse -Force
    }
}

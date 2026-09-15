[CmdletBinding()]
param(
    [string]$OutputPath,
    [string]$GeneratorSpec = "examples/module-starter/collection.yaml"
)

$ErrorActionPreference = 'Stop'
$results = @()

function Invoke-TimedStep {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][scriptblock]$Action
    )

    $watch = [Diagnostics.Stopwatch]::StartNew()
    & $Action
    $exitCode = $LASTEXITCODE
    $watch.Stop()
    $script:results += [pscustomobject]@{
        name = $Name
        duration_ms = [math]::Round($watch.Elapsed.TotalMilliseconds)
        exit_code = $exitCode
    }
    if ($exitCode -ne 0) {
        throw "$Name failed with exit code $exitCode"
    }
}

Invoke-TimedStep -Name 'targeted-ui-contract' -Action { go test ./internal/ui ./tests/contract }
Invoke-TimedStep -Name 'full-compile' -Action { go test ./... -run '^$' }
$benchmarkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("hydrogo-dx-" + [guid]::NewGuid().ToString('N'))
$projectParent = Join-Path ([System.IO.Path]::GetTempPath()) ("hydrogo-project-" + [guid]::NewGuid().ToString('N'))
try {
    Invoke-TimedStep -Name 'first-project-scaffold' -Action {
        go run ./cmd/hydro new velocity-smoke --root $projectParent --preset=news --database=sqlite --profile=compact --no-demo
    }
    Invoke-TimedStep -Name 'first-resource-generation' -Action {
        go run ./cmd/hydro generate module --spec $GeneratorSpec --out $benchmarkRoot
    }
    Invoke-TimedStep -Name 'generated-compatibility-check' -Action {
        go run ./cmd/hydro validate generator --spec $GeneratorSpec --root $benchmarkRoot
    }
}
finally {
    if (Test-Path -LiteralPath $benchmarkRoot) {
        Remove-Item -LiteralPath $benchmarkRoot -Recurse -Force
    }
    if (Test-Path -LiteralPath $projectParent) {
        Remove-Item -LiteralPath $projectParent -Recurse -Force
    }
}

$report = [pscustomobject]@{
    generated_at = (Get-Date).ToUniversalTime().ToString('o')
    commit = (git rev-parse HEAD).Trim()
    go_version = (go version).Trim()
    generator_spec = $GeneratorSpec
    acceptance_threshold_minutes = 15
    steps = $results
}
$json = $report | ConvertTo-Json -Depth 4
if ($OutputPath) {
    $parent = Split-Path -Parent $OutputPath
    if ($parent -and -not (Test-Path -LiteralPath $parent)) {
        New-Item -ItemType Directory -Path $parent -Force | Out-Null
    }
    Set-Content -LiteralPath $OutputPath -Value $json -Encoding UTF8
}
$json

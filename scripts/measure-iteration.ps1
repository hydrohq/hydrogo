[CmdletBinding()]
param(
    [string]$OutputPath,
    [string]$GeneratorSpec = "examples/module-starter/collection.yaml",
    [ValidateRange(1, 10)]
    [int]$Runs = 3
)

$ErrorActionPreference = 'Stop'
$results = @()
$script:currentRun = 0

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
        run = $script:currentRun
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

for ($run = 1; $run -le $Runs; $run++) {
    $script:currentRun = $run
    $benchmarkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("hydrogo-dx-" + [guid]::NewGuid().ToString('N'))
    $projectParent = Join-Path ([System.IO.Path]::GetTempPath()) ("hydrogo-project-" + [guid]::NewGuid().ToString('N'))
    try {
        Invoke-TimedStep -Name 'first-project-scaffold' -Action {
            go run ./cmd/hydro new velocity-smoke --root $projectParent --preset=news --database=sqlite --profile=compact --no-demo
        }
        Invoke-TimedStep -Name 'first-route-contract' -Action {
            $manifest = Join-Path $projectParent "velocity-smoke\e2e\manifest.json"
            if (-not (Test-Path -LiteralPath $manifest)) {
                throw "generated frontend route manifest is missing: $manifest"
            }
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
}

$acceptanceNames = @('first-project-scaffold', 'first-route-contract', 'first-resource-generation', 'generated-compatibility-check')
$runSummaries = @()
for ($run = 1; $run -le $Runs; $run++) {
    $runResults = @($results | Where-Object { $_.run -eq $run -and $_.name -in $acceptanceNames })
    $duration = [math]::Round((($runResults | Measure-Object -Property duration_ms -Sum).Sum))
    $runSummaries += [pscustomobject]@{
        run = $run
        duration_ms = $duration
        passed = ($runResults.Count -eq $acceptanceNames.Count -and @($runResults | Where-Object { $_.exit_code -ne 0 }).Count -eq 0)
    }
}
$orderedDurations = @($runSummaries | ForEach-Object { [double]$_.duration_ms } | Sort-Object)
$medianDuration = $orderedDurations[[math]::Floor(($orderedDurations.Count - 1) / 2)]

$report = [pscustomobject]@{
    generated_at = (Get-Date).ToUniversalTime().ToString('o')
    commit = (git rev-parse HEAD).Trim()
    go_version = (go version).Trim()
    generator_spec = $GeneratorSpec
    runs = $Runs
    acceptance_threshold_minutes = 15
    acceptance = [pscustomobject]@{
        measured_steps = $acceptanceNames
        median_duration_ms = $medianDuration
        min_duration_ms = ($orderedDurations | Select-Object -First 1)
        max_duration_ms = ($orderedDurations | Select-Object -Last 1)
        passed = (@($runSummaries | Where-Object { -not $_.passed }).Count -eq 0 -and $medianDuration -lt (15 * 60 * 1000))
    }
    run_summaries = $runSummaries
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

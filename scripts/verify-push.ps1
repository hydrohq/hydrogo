$ErrorActionPreference = "Stop"

function Invoke-Required {
    param([string]$Name, [scriptblock]$Action)
    Write-Host "==> $Name"
    & $Action
    if ($LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE" }
}

Write-Host "HydroGo fast push verification"
Invoke-Required "module integrity" { go mod verify }
Invoke-Required "gofmt" {
    $trackedGo = @(git ls-files "*.go")
    if ($LASTEXITCODE -ne 0) { throw "git ls-files failed" }
    $unformatted = @($trackedGo | ForEach-Object { gofmt -l $_ } | Where-Object { $_ })
    if ($unformatted) { throw "gofmt required: $($unformatted -join ', ')" }
}
Invoke-Required "go vet" { go vet ./... }
Invoke-Required "test suite" { go test ./... -p 4 -count=1 -timeout=3m }
Invoke-Required "hydro build" { go build -trimpath -buildvcs=false -o .tmp\local\hydro-push.exe ./cmd/hydro }

Write-Host "PASS: fast push verification completed."
Write-Host "Full release/security verification: HYDRO_PUSH_FULL=1 git push"

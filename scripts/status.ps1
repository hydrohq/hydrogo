[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"
$taskFile = Join-Path $PSScriptRoot "..\docs\PROFESSIONALIZATION_MASTER_TASKS.md"
if (-not (Test-Path -LiteralPath $taskFile)) {
    throw "Task list not found: $taskFile"
}

$done = (Select-String -LiteralPath $taskFile -Pattern '- \[x\]').Count
$open = (Select-String -LiteralPath $taskFile -Pattern '- \[ \]').Count
$partial = (Select-String -LiteralPath $taskFile -Pattern '- \[~\]').Count
$head = (git rev-parse --short HEAD).Trim()
$branch = (git branch --show-current).Trim()
$status = @(git status --short)
$sync = "unknown"
try {
    $remote = (git rev-parse --abbrev-ref --symbolic-full-name '@{u}' 2>$null).Trim()
    if ($remote) {
        $ahead = [int](git rev-list --count "$remote..HEAD")
        $behind = [int](git rev-list --count "HEAD..$remote")
        $sync = "ahead=$ahead behind=$behind"
    }
} catch {
    $sync = "no-upstream"
}

Write-Output "HydroGo status"
Write-Output "branch=$branch commit=$head sync=$sync"
Write-Output "tasks done=$done open=$open partial=$partial total=$($done + $open + $partial)"
if ($status.Count -eq 0) {
    Write-Output "worktree=clean"
} else {
    Write-Output "worktree=dirty files=$($status.Count)"
    $status | ForEach-Object { Write-Output "  $_" }
}

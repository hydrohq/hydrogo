# R15 developer iteration baseline (after closing CS2) — 2026-09-17

The same `scripts/measure-iteration.ps1` workflow was repeated after the
background CS2 process was closed. YouTube remained open, so this is a cleaner
run but not a laboratory-isolated measurement.

- Revision: `adec874841fa7903fb1a64bd23d3f78f0dc5b494`
- Go: `go1.27.0 windows/amd64`
- Acceptance flow: **1.289 seconds** (all steps passed)
- Targeted UI/contract tests: 0.603 s
- Full compile: 2.560 s

Compared with the previous run (2.691 s acceptance flow), this run was faster,
which confirms that background load can materially affect local timing. The
machine-readable result is in
`R15-dx-iteration-baseline-after-game-2026-09-17.json`.

This remains a developer-experience baseline, not a production capacity/SLA
claim. Two independent developer/customer environments and cold-cache repeats
are still needed for the comparative roadmap gate.

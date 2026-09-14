# R5 validation: race suite

The complete repository passes Go's race detector in the WSL development
toolchain (`gcc` and Go are available there):

```bash
cd /mnt/c/dev/cms
CGO_ENABLED=1 go test -race ./...
```

The run completed successfully across all packages on 2026-09-14. On Windows,
`-race` requires a C compiler; use the WSL command above when `gcc` is not on
`PATH`. This is a concurrency signal for local development, not a substitute
for production capacity and soak testing.

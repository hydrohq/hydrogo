# R18 WSL FFmpeg media derivative contract — 2026-09-17

WSL Ubuntu exposes `/usr/bin/ffmpeg`. The real transformer was exercised with a
deterministic 1×1 PNG and a single-threaded WebP conversion.

```bash
go test ./internal/media \
  -run '^TestFFmpegTransformerProducesWebPWhenEncoderIsAvailable$' \
  -count=1
```

Result: **PASS** (`0.245 s`). The test verifies that FFmpeg is discovered from
PATH, the WebP file is created in the bounded media directory, the returned key
is the generated filename, the MIME type is `image/webp`, and the output is
non-empty.

This is an encoder availability and wiring check only. It does not certify large
video workloads, codec coverage, RAM/CPU limits, queue throughput, or production
retention/object storage behavior.

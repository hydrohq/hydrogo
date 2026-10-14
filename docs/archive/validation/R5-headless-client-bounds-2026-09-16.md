# R5 validation: headless client bounds

Date: 2026-09-16

- `npm run check` and `npm run build` pass in `examples/headless-starter`.
- The client rejects timeout values outside `1..120000` ms.
- Requests abort after the configured timeout even when a caller supplies an
  `AbortSignal`; captured error text is capped at 8,192 code points.
- The SDK remains dependency-free and accepts an injected `fetch` for tests or
  host-specific transport.

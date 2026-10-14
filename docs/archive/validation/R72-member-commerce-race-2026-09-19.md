# R72 member commerce race validation

**Date:** 2026-09-19  
**Status:** WSL/Linux race suite passed for member, commerce, providers,
transport, SQL store and contract packages.

The Windows host cannot run Go race mode because GCC is not installed; the
supported WSL path has Go 1.27 and `/usr/bin/gcc` and completed the full scoped
race command without reports:

```text
CGO_ENABLED=1 go test -race -p 1 -count=1 -timeout=5m \
  ./modules/members/... ./modules/commerce/... ./tests/contract
```

This is local concurrency evidence. It does not certify multi-process Redis,
reverse-proxy, provider callback, restore or customer production behavior.

# R2 Pages and media output validation

**Date:** 2026-09-08  
**Roadmap:** R2 Pages and initial customer contact  
**Status:** Success; R1 S03 Quill/HTMX experiment is a separate open gate.

## Validated behaviors

- SQLite and PostgreSQL migration contracts run the same Pages draft/revision/publish flow.
- A draft update does not change the published projection; publish generation, audit, and outbox progress within the same transaction.
- Two drafts can be created for the same locale/slug; only the first successful publish route wins, and the losing one returns a conflict.
- An update made with a stale editor version returns a conflict.
- The sanitizer is applied before writing to the revision body.
- Media upload metadata has a `pending → ready` transition; non-ready assets are not served.
- JPEG/PNG/WebP uploads are fully decoded in addition to the size header; truncated or corrupt images are not committed to immutable storage.
- Private media returns 401 for unauthenticated requests, public ready media responds successfully.
- In the event of an injected outbox error, page, revision, and audit records roll back together.

## Executed checks

```text
go test ./...
go vet ./...
go build ./cmd/hydro
cd examples/client-edition && go build ./...
```

The same checks were run on GitHub CI with SQLite and PostgreSQL contract services. Migration metadata checksums and fixture isolation were validated.

## Open limitations

- SEO, restore, blue/green, MFA, and commercial Compact capacity evidence within the R3 scope are outside the scope of this report. The post-publish public cache generation guard was validated in a separate S05-C progress commit.
- Completion of the actual Quill/HTMX asset and CSP experiment is required for R1 S03.

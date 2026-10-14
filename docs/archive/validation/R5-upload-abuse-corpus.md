# R5 validation: upload abuse corpus

Date: 2026-09-14

- `go test ./internal/security ./internal/media` passes.
- Coverage includes unsafe redirect/host confusion, request and multipart body
  limits, path traversal, disallowed MIME, corrupt/truncated images, and pixel
  bombs.
- Responses fail closed with bounded public errors; storage paths and parser
  details are not exposed.
- External penetration testing and malware-scanning provider validation remain
  release gates.

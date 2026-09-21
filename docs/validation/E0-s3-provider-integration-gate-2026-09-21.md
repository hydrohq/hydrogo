# E0 S3-Compatible Provider Integration Gate — 2026-09-21

HydroGo now has an opt-in live-provider test and runner:

```text
scripts/verify-s3-provider.ps1
```

The gate requires operator-supplied `HYDRO_TEST_S3_ENDPOINT`, access key,
secret key, and bucket (with optional region, TLS, and path-style settings).
`TestS3ProviderIntegration` then uses the real configured provider to perform:

- object PUT into the quarantine namespace;
- GET and byte-for-byte round-trip verification;
- one-minute presigned GET over HTTP;
- cleanup with DELETE.

Without those credentials the test is skipped by `go test`, while the runner
fails closed and explicitly reports the missing environment. No provider is
claimed certified by local mock tests; a passing report requires the operator
to run this gate against the controlled S3/MinIO environment.

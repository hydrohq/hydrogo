# ADR-0144: Customer package source exclusion

- **Status:** Accepted
- **Supersedes:** None

Customer packaging is allowlist-based. The build scripts produce the runtime from the vendor checkout, then copy only deployment data and Client Edition files. They reject source directories, Go module metadata, VCS metadata, and signing materials. Docker's final image follows the same boundary and never copies build-stage migration source.

The runtime embeds Core and active module behavior in the binary. This is source exclusion, not a promise that a privileged operator cannot inspect a running binary.

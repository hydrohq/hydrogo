# R6 erasure/storage GC validation

`internal/erasure.GC` performs a pin check right before deleting each storage key; a pinned key is skipped, deletion stops upon a provider error, and the list of deleted keys is deterministically returned. The provider delete operation must be idempotent.

This is a process-local orchestration contract. The actual S3-compatible provider, distributed atomicity between pin and delete, crash recovery, and GC race load testing remain open.

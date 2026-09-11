# R3 external erasure intent log validation

**Date:** 2026-09-08  
**Status:** Compact file adapter prototype, hash-anchored checkpoint replay, and SQLite Pages/media SQL adapter succeeded; provider/key recovery drill remains open.

The append-only JSONL adapter in `internal/erasure` stores erasure intents outside the DB. Each record carries the edition, fixed target, operation type, sequence, and previous hash link. File writes are completed with `Sync`; the same operation ID is idempotent and cannot be reused with a different target. `Verify` rejects sequence gaps, hash chain errors, corrupted JSON, and record field errors.

`EntriesAfter` returns intents after the checkpoint to be replayed from the verified chain; the idempotent application of content and media purge operations is the responsibility of the upper layer.

After verifying the chain, `Replay` sequentially feeds intents to the handler. In case of a handler error, the checkpoint is not advanced; re-running with the same initial checkpoint is validated in the contract test. `SQLAdapter` idempotently deletes revision and page records for `erase.page`, and media metadata for `erase.media` within a transaction; applying the same log checkpoint a second time does not yield an error.

`ReplayFromCheckpoint` no longer accepts only a sequence number; it validates the hash anchor of the persisted checkpoint against the current chain. In the event of log truncate/replacement or an invalid old checkpoint, the replay is rejected and operator intervention is required before opening external traffic. On the backup side, `PruneSnapshots` validates the newest/pinned retention behavior on local snapshots with a valid manifest, and the fail-closed rule on a corrupted manifest; the external provider GC is not yet certified.

Executed check:

```text
go test ./internal/erasure
```

Remaining S07 tasks are the external backup provider, key recovery, retention, storage garbage-collection, and clean environment restore replay drill.

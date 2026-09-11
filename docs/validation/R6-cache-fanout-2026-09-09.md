# R6 cache fan-out validation

`internal/cache.Fanout` uses a bounded subscriber buffer. When a slow subscriber overflows with messages, the queue is not grown; a single resync signal is generated and the monotonic generation is preserved. The subscriber cannot clear its resync state without validating the current generation snapshot via `Reconcile`. A closed bus closes the subscriber channels and does not advance the generation of a new publish.

This contract validates process-local behavior. Actual multi-web/worker, L2 provider, generation snapshot reconciliation post-reconnect, and network partition load testing remain open.

`internal/cache.TwoTier` prioritizes L1 hits, warms up L1 on an L2 hit, and falls back to a bounded local loader path on L2 get/set errors. The actual connection pool, partition, and multi-instance invalidation testing of the remote backend are outside the scope of this contract.

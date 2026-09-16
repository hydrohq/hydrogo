# ADR-0117: Bound headless client waits and error bodies

- **Status:** Accepted
- **Date:** 2026-09-16

The headless starter applies a 15-second default request timeout, configurable
only from 1 ms to 120 seconds, and caps captured error response text at 8,192
code points. Caller-provided abort signals are forwarded into the same bounded
controller, so explicit cancellation never disables the timeout. The client
remains dependency-free and lets callers provide a custom fetch implementation.
This protects browser applications from hanging requests and unbounded error
payloads without deciding presentation copy in the SDK.

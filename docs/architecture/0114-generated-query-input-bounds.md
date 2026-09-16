# ADR-0114: Bound generated frontend query inputs

- **Status:** Accepted
- **Date:** 2026-09-16

Generated frontend query helpers are the most convenient path for Client
Edition developers, so they must be safe by default. The generator now emits
guards for positive integer pages, page sizes up to 100, bounded sort names,
at most 32 filters, allowlisted filter-key syntax, and 256-codepoint filter
values. Invalid input fails before a request URL is built.

The limits mirror the provider-neutral admin query contract and remain
framework-independent; applications may add stricter module limits.

# ADR-0115: Generate deterministic kit descriptors

- **Status:** Accepted
- **Date:** 2026-09-16

Capability kits are optional module contracts. Their generated frontend
metadata now includes a stable descriptor (`name`, resolved dependencies,
`optional`) in addition to the backwards-compatible kit-name tuple. The
descriptor contains no provider implementation or infrastructure details, and
omitted kits remain absent from generated output.

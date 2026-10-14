# ADR-0023: Provider-agnostic design systems

- **Status:** Accepted
- **Date:** 2026-09-15
- **Related decisions:** [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0020](0020-headless-first-admin-studio-and-frontend-templates.md)

## Decision

HydroGo themes expose semantic slots, tokens and component intent through the
versioned theme manifest. A `design_system` identifier is metadata, not a hard
dependency: `shadcn`, Material, Carbon, Bootstrap, a customer system or a
fully custom implementation can satisfy the same slots.

Core and modules render semantic UI primitives and never import a design-system
package. The selected theme/provider maps those primitives to its own CSS,
markup and interaction conventions. A provider may be dependency-free,
server-rendered, or backed by an edition-specific client bundle, provided its
assets are declared and its accessibility/security contracts pass.

## Consequences

Existing HydroGo and shadcn modes remain compatible. Adding a new design system
does not fork Core or force every customer to ship its assets. Provider-specific
features belong in an adapter/Client Edition and must degrade to the semantic
primitive when unavailable.

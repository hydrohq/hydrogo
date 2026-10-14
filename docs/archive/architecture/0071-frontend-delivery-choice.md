# ADR-0071: Choose frontend delivery by demonstrated boundary needs

## Status

Accepted

## Decision

Server-rendered HTML is the default for SEO/public content, Admin Studio, and
rapid prototypes. Fragment updates are allowed only when they preserve the
same semantic and security boundary. A separately deployed frontend requires
the versioned API/SDK and an explicit session/CORS compatibility decision.

## Consequences

Teams get a fast default without creating a second application architecture.
Product, deployment, and API ownership decisions remain visible before a
separate frontend is introduced.

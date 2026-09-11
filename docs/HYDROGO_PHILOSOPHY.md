# Hydro Manifesto

> **Fluid as water. Resilient as rock. Control always remains with us.**

If a project's technical architecture is its skeleton, its philosophy is its soul. Code is refactored over time, libraries and infrastructures evolve; HydroGo's core decision-making principles safeguard the product's purity, robustness, and the commitments made to customers.

HydroGo adapts to diverse scales and industries. This adaptability is not formlessness: security, data sovereignty, explicit contracts, and operability are immutable attributes.

## 1. Be Like Water — Fluidity and Adaptability

> The system does not resist the vessel; it takes form according to necessity.

HydroGo adapts with distinct profiles to the lightness of a corporate website, the high-throughput read traffic of a news platform, the API demands of a mobile application, and the rigor of transaction-heavy systems.

Adaptability does not mean carrying every feature into every deployment. An inactive module does not incur route, migration, asset, worker, or infrastructure overhead.

**Non-negotiable rule:** Unused capabilities must never be imposed on a customer's resources or operational burden.

## 2. Ownership and Independence — Sovereignty

> If you cannot control your infrastructure, you cannot control your direction.

HydroGo is not a fork of any third-party CMS. The product's source direction, build chain, distribution format, and data portability remain under our full control. Customers retain complete authority over their data, backups, and runtime environment; an exit strategy from the platform is always available.

Independence does not mean reinventing everything. Reliable dependencies are selected intentionally, their versions and licenses pinned, and isolated behind replaceable boundaries.

**Non-negotiable rule:** We never surrender data, build, or deployment control to the mercy of a single external provider.

## 3. Verify First, Automate Later — Pragmatism

> Do not abstract what you haven't run; do not automate what you haven't observed.

First, a real customer flow is developed manually and end-to-end. Service boundaries, data models, authorization, admin UX, and operational behaviors are validated in the field. The CLI and schema generators automate only patterns with proven repetition.

A generator never guesses business logic. It generates standard CRUD and presentation scaffolding; customer-specific rules are written explicitly by engineers at clear extension points.

**Non-negotiable rule:** A pattern does not become a Core contract until verified across two distinct real-world use cases.

## 4. Secure by Design & Default Deny

> An open door is not merely an error; it is a fundamental design flaw that must be prevented.

Security is not left to a developer remembering the right detail every time. Route classification, record-level authorization, type safety, automatic escaping, sanitizers, CSRF protection, and secure sessions are enforced by default.

Templ facilitates safe output generation; it is not, by itself, an absolute XSS guarantee. `templ.Raw`, URLs, rich text, and third-party client code remain subject to strict security contracts.

**Non-negotiable rule:** The secure path must always be the default, visible, and most ergonomic development path.

## 5. Rock-Solid Simplicity, Lightning Speed — Minimalist Power

> Perfection is achieved not through fewer lines of code, but through fewer surprises and zero unnecessary overhead.

Every layer, package, and dependency earns its presence with clear, single-purpose responsibility. Performance claims are measured under defined hardware, traffic, data volume, and cache conditions. Small and distributed deployments share the same Core contracts; only the capacity and infrastructure profile varies.

HTTP cache, local memory, distributed cache, databases, and CDNs are not competitors; each layer is activated only when required.

**Non-negotiable rule:** Complexity is added only if it solves a measured necessity and simplifies the total system.

## 6. Stable Core, Free Customer — Backwards Compatibility

> As the trunk expands, branches must not break unexpectedly.

The boundary between HydroCore and Client Editions is explicit and versioned. Customer customization is never performed as a HydroCore fork. It progresses via theme tokens, module contracts, view slots, and application service extension points.

Published contracts within the same major version are preserved. Unavoidable breaking changes require a new major version, a deprecation grace period, migration tooling, and a documented upgrade path. Zero-downtime deployment is protected by the expand–migrate–contract discipline, enabling temporary coexistence of old and new versions.

**Non-negotiable rule:** A Core change cannot be published without passing contract tests across supported customer editions.

## Engineering Decision Filter

When a new technology, layer, or feature is proposed, the following questions must be answered:

1. Does it enhance the customer's data and deployment sovereignty?
2. Can it be completely deactivated with zero footprint in the Compact profile?
3. Is its behavior across certified databases and future adapters clearly defined?
4. Are cache invalidation and failure modes explicit?
5. Does it affect the interoperability of old and new versions?
6. Can the customer solve their problem without forking Core?
7. Has it been validated through real-world usage or empirical benchmarks?

If any of these questions shifts the architectural direction, the decision must be recorded as an ADR.

## The Manifesto in Brief

We adapt fluidly like water, carrying no unused burden.  
We retain sovereignty over our data, builds, and deployment trajectory.  
We verify in real-world scenarios before automating.  
We make security the system default, not an afterthought.  
We measure performance and rigorously justify complexity.  
We maintain a stable Core and empower the customer.

---

**HydroGo:** A CMS platform that adapts to every scale without compromising on security, data ownership, or explicit contracts.

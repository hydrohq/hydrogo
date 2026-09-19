# Admin Studio extension contract

HydroGo does not support loading arbitrary Go `.so`/`.dll` plugins into the
closed binary. The supported customer path is an independently deployed admin
surface plus an external event consumer.

An extension package contains:

1. A signed JSON manifest verified against operator-configured Ed25519 public
   keys by `extension/v1.ParseSignedAdminExtensionManifest`.
2. An HTTPS entrypoint and relative surface paths rendered by the host in an
   isolated origin/iframe.
3. Explicit permission labels used for host-side authorization checks; the
   manifest cannot grant itself access.
4. Optional event subscriptions delivered through HydroGo's signed webhook
   boundary. The extension owns its UI, database and ERP/XML logic.

The first contract implementation is deliberately declarative. It does not yet
promise arbitrary Templ injection, same-origin JavaScript, direct SQL access,
or runtime Go code loading. Those choices would expand the trusted computing
base and break the binary portability goal.

## Example manifest

```json
{
  "id": "acme-reports",
  "name": "Acme Reports",
  "version": "1.0.0",
  "entrypoint_url": "https://reports.acme.example/admin",
  "surfaces": [{
    "id": "sales", "label": "Sales report", "path": "/sales",
    "placement": "reports", "permissions": ["reports.read"]
  }],
  "subscriptions": [{"event": "order.created"}]
}
```

The host must still add CSP/frame policy, lifecycle/rollback and a versioned
postMessage protocol before enabling customer-provided manifests in a production
tenant. Private signing keys remain in the release environment; a key embedded
in a customer manifest is never trusted by itself.

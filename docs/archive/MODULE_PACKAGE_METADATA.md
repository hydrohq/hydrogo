# Signed and allowlisted module package metadata

This is a design boundary, not a marketplace. A module package may carry a
metadata document with:

- `name`, `version`, and SDK major compatibility;
- exact module dependencies and capability declarations;
- migration IDs and asset digests;
- source revision, build toolchain, and package digest;
- an Ed25519 signature over the canonical metadata plus package digest.

Installation accepts only names in the deployment's explicit module allowlist,
the expected SDK major, and a signature matching a separately distributed
trusted public key. Unknown capabilities, duplicate dependencies, missing
digests, and mismatched package contents fail closed.

The runtime does not download or load arbitrary Go plugins. Package selection
still creates a reviewed release; inactive modules create no routes, workers,
migrations, or assets. Production key ceremony, key rotation, registry
attestation, and legal licensing policy remain external gates.

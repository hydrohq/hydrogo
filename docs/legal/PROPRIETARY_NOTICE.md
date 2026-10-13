# HydroGo Open Source & Licensing Notice

> [!NOTE]
> HydroGo is open-source software distributed under the [MIT License](../../LICENSE).

This repository was transitioned from a private/commercial prototype model to open-source software under the MIT License. All source code within this repository is governed by the terms of the MIT License.

## Cryptographic Licensing Subsystem Context
The codebase contains optional cryptographic offline licensing primitives (`internal/licensing`, `hydro.lic`) originally engineered for air-gapped commercial deployments and Edition boundaries. In the open-source distribution:
- Development and open-source deployments run freely without license restrictions (using `--dev` or `HYDRO_DEV_MODE=1` where applicable, or standard open-source builds).
- The cryptographic verification primitives remain available in the repository as reusable architectural components for organizations that build custom closed-source or enterprise distributions on top of the open-source Core.
- See the root [`LICENSE`](../../LICENSE) for governing legal terms.

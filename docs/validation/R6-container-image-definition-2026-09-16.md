# R6 container image definition validation

The repository now includes a reproducible, credential-free OCI image
definition in `Dockerfile` and a bounded build context in `.dockerignore`.
The image compiles a static Linux amd64 binary, embeds only public runtime
assets, runs as the non-root `hydro` user, and requires deployment-owned config
and license material to be mounted at runtime.

Local evidence:

```powershell
pwsh -File scripts/container-lint.ps1
```

Result: `PASS: container image definition is non-root, credential-free, and
reproducible`.

The local host does not have a Docker/OCI daemon, so an actual image build,
registry push, vulnerability scan, and runtime healthcheck remain deployment
environment gates. The production checklist intentionally remains open until
those external proofs exist.

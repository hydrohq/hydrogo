# Developer Integration for security-check

For HydroGo, the `ersinkoc/security-check` bundle is installed in the local Codex skill directory.
The installation source commit:

```text
80d68f68da642f65883ca1791f03fb3ceaf456d5
```

## Codex

After installing the global skill, the following prompts work directly in a new Codex turn:

```text
run security check
scan diff
```

## Antigravity, agy, and Other Agentskills CLIs

The official upstream method to install in the project root for these tools:

```bash
npx skills add ersinkoc/security-check
```

After installation, the `security-check` orchestrator and `sc-*` skills should appear under `.agents/skills/`.
Windows PowerShell alternative:

```powershell
irm https://raw.githubusercontent.com/ersinkoc/security-check/main/skills.ps1 | iex
```

Record the source commit and checksum within your team during installation; do not automatically trust
the remote `main` branch directly in production build pipelines. The skill must only perform source code
reviews and explicitly permitted isolated local checks; scanning live endpoints, credentials, or shared
infrastructure requires separate authorization and staging gates.

HydroGo's latest source-only audit report is located under [`security-report/`](../security-report/).
Do not treat this result as a substitute for a live production penetration test.

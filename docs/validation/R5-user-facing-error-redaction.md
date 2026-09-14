# R5 validation: user-facing error redaction

The member transport regression suite renders the login page and verifies that
browser storage markers and runtime error details are absent. It also exercises
the response sanitizer directly and requires the stable generic transport
message.

Run:

```powershell
go test ./modules/members/transport
```

The full local verification hook additionally checks formatting, static
analysis, all tests, fuzz smoke tests, migrations, and release asset guards.

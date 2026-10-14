# Test stability policy

The ordinary CI suite proves one clean execution. The stability runner repeats
the highest-risk packages with a fresh shuffled order to detect order-dependent
or timing-dependent failures. It does not retry failures: a failed iteration is
recorded as a failure.

Run locally with `HYDRO_TEST_STABILITY_RUNS=3 bash scripts/measure-test-stability.sh`
or set the same environment variable in PowerShell and run
`scripts/measure-test-stability.ps1`. The generated JSON evidence is a CI
artifact, not a committed production-certification claim.

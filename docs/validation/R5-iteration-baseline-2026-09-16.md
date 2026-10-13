# R5 — HydroGo developer iteration measurement

Command: `./scripts/measure-iteration.ps1 -OutputPath .tmp/iteration-baseline-2026-09-16.json`

Environment: Windows amd64, Go 1.27.0, commit `777cb3acc3a1edc5900b8ee77898b236529ac95e`.

| Step | Duration | Result |
| --- | ---: | --- |
| targeted-ui-contract | 2.401 s | PASS |
| full-compile | 10.191 s | PASS |
| first-project-scaffold | 0.465 s | PASS |
| first-route-contract | 0.030 s | PASS |
| first-resource-generation | 0.380 s | PASS |
| generated-compatibility-check | 0.550 s | PASS |

The first running project/resource acceptance workflow totaled **1.425 s**, passing the 15-minute local acceptance threshold. This represents HydroGo's single-machine baseline only; it does not claim relative speed superiority over Payload without two independent scenarios and real customer hardware.

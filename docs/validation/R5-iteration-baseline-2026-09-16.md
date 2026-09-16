# R5 — HydroGo geliştirici iterasyon ölçümü

Komut: `./scripts/measure-iteration.ps1 -OutputPath .tmp/iteration-baseline-2026-09-16.json`

Ortam: Windows amd64, Go 1.27.0, commit `777cb3acc3a1edc5900b8ee77898b236529ac95e`.

| Adım | Süre | Sonuç |
| --- | ---: | --- |
| targeted-ui-contract | 2.401 s | PASS |
| full-compile | 10.191 s | PASS |
| first-project-scaffold | 0.465 s | PASS |
| first-route-contract | 0.030 s | PASS |
| first-resource-generation | 0.380 s | PASS |
| generated-compatibility-check | 0.550 s | PASS |

İlk çalışan proje/resource kabul akışının toplamı **1.425 s** olarak ölçüldü;
15 dakikalık yerel kabul eşiği geçildi. Bu yalnızca HydroGo'nun tek makinedeki
baseline'ıdır; Payload karşılaştırması, iki bağımsız senaryo ve gerçek müşteri
donanımı olmadan göreli hız iddiası değildir.

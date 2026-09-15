# R5 — HydroGo geliştirici iterasyon ölçümü

Komut: `./scripts/measure-iteration.ps1 -OutputPath .tmp/iteration-baseline-2026-09-16.json`

Ortam: Windows amd64, Go 1.27.0, commit `a31587f9e0224019184a4571a7cf94a218fe0f1b`.

| Adım | Süre | Sonuç |
| --- | ---: | --- |
| targeted-ui-contract | 4.786 s | PASS |
| full-compile | 18.071 s | PASS |
| first-project-scaffold | 20.500 s | PASS |
| first-route-contract | 0.044 s | PASS |
| first-resource-generation | 1.909 s | PASS |
| generated-compatibility-check | 1.001 s | PASS |

İlk çalışan proje/resource kabul akışının toplamı **23.454 s** olarak ölçüldü;
15 dakikalık yerel kabul eşiği geçildi. Bu yalnızca HydroGo'nun tek makinedeki
baseline'ıdır; Payload karşılaştırması, iki bağımsız senaryo ve gerçek müşteri
donanımı olmadan göreli hız iddiası değildir.

# Dokümantasyon bağlantı denetimi — 2026-09-16

`docs/**/*.md` içindeki göreli Markdown bağlantıları tarandı. HTTP/mailto
bağlantıları dışarıda bırakılarak her yerel hedef dosya sistemi üzerinde kontrol
edildi.

```text
MISSING_COUNT=0
```

İki kırık ADR bağlantısı mevcut ADR-0065 ve ADR-0012 hedeflerine yönlendirildi.
Bu denetim içerik doğruluğunu veya harici URL erişilebilirliğini garanti etmez;
yalnızca yerel doküman hedeflerinin mevcut olduğunu kanıtlar.

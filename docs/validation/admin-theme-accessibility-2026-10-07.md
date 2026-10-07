# Admin Studio tema erişilebilirlik matrisi

**Tarih:** 2026-10-07  
**Sonuç:** 112/112 Admin route × tema/mod taraması geçti  
**Kapsam:** Admin dashboard, sayfa/haber oluşturma, medya, kullanıcılar,
izinler, güvenlik, audit, yedek, control, operations, developer ve rapor
yüzeyleri; HydroGo, shadcn, Nova, Aurora temalarının açık ve koyu modları.

`e2e/custom/admin-management.spec.ts` içindeki gerçek Edition Chromium testi
her kombinasyonda rotayı yeniden açar, seçilen tema/modu doğrular ve axe-core
ile WCAG 2.1 A/AA kurallarını tarar. Doğrulama komutu:

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project=chromium --workers=1 \
  --grep 'editor and viewer sessions|admin login opens the Edition management areas|admin routes meet WCAG' \
  --reporter=line
```

İlk koşular, yalnız varsayılan HydroGo temasında görünmeyen kontrast kusurlarını
ortaya çıkardı. Shadcn/Aurora yeşil ve ikincil metin tonları, Shadcn/Nova/Aurora
amber tonu, Nova ikincil metin tonu ve içerik etiketi rengi düzeltildi. Son
koşuda 3 test geçti; 14 rota × 8 tema/mod birleşiminde axe ihlali kalmadı.

Bu Chromium/axe kanıtı fiziksel cihaz veya WebKit görsel parity testi değildir.
Generated liste/formun kendi 8/8 matrisi ayrı kapsamdır; public ve tüm generated
sayfa türleri bu matrise dahil değildir.

# Admin permission override doğrulaması

**Tarih:** 2026-10-07  
**Durum:** Tamamlandı  
**Kapsam:** Tek Edition admin'in kullanıcı başına içerik yazma override'ı; editor browser oturumuna etkisi.

Admin yönetim E2E akışında admin, `editor@example.test` hesabı için
`admin.content.write` değerini `deny` yaptı. Sayfa yenilendiğinde deny seçimi
korundu. Ayrı Chromium context'inde gerçek editor hesabıyla giriş yapıldı ve
`GET /admin/pages/new` ile `GET /admin/pages` istekleri 403 döndürdü; create
formu gösterilmedi. Yeni session'dan alınan geçerli CSRF tokenıyla
`POST /admin/pages` de 403 döndürdü; mutation reddi CSRF katmanından bağımsız
olarak doğrulandı. Eski editor session'ı `/auth/me` üzerinden 401 aldı. Admin
override'ı `allow` yaptı
ve sayfa yenilendikten sonra seçili değer korundu. Normal editor yazma akışı
aynı test dosyasının role-boundary senaryosunda ayrıca doğrulanır. Deny'nin
etkin route iznine ve kullanıcı session yaşam döngüsüne etkisi gözlemlendi.

## Komut ve sonuç

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project=chromium --workers=1 --reporter=line
```

Permission deny/allow ve rol sınırı için doğru seri fixture sırasıyla son
hedefli koşu: **2/2 geçti**. Önceki tam dosya koşusu 5/5 idi; yeni eklenen
geçerli-CSRF mutation assertion'ı bu hedefli koşuda tekrarlandı. Senaryo gerçek
SQLite Edition server fixture'ında çalışır.

Bu doğrulama `admin.content.write` override'ının etkin route davranışını ve
session iptalini, geçerli-CSRF mutation dahil, kapsar. Her permission catalog
entry için her rol ve her mutation varyantının tam matrisi henüz planın Faz 4
açık kapsamındadır.

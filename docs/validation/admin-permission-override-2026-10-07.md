# Admin permission override doğrulaması

**Tarih:** 2026-10-07  
**Durum:** Tamamlandı  
**Kapsam:** Tek Edition admin'in kullanıcı başına içerik yazma override'ı; editor browser oturumuna etkisi.

Admin yönetim E2E akışında admin, `editor@example.test` hesabı için
`admin.content.write` değerini `deny` yaptı. Sayfa yenilendiğinde deny seçimi
korundu. Ayrı Chromium context'inde gerçek editor hesabıyla giriş yapıldı ve
`GET /admin/pages/new` isteği 403 döndürdü; create formu gösterilmedi. Eski
editor session'ı `/auth/me` üzerinden 401 aldı. Admin override'ı `allow` yaptı
ve sayfa yenilendikten sonra seçili değer korundu. Normal editor yazma akışı
aynı test dosyasının role-boundary senaryosunda ayrıca doğrulanır. Deny'nin
etkin route iznine ve kullanıcı session yaşam döngüsüne etkisi gözlemlendi.

## Komut ve sonuç

```sh
cd e2e
npx playwright test custom/admin-management.spec.ts --project=chromium --workers=1 --reporter=line
```

Son doğrulanan tam dosya koşusu: **3/3 geçti.** Bu dosyadaki permission deny/allow senaryosu; role
boundary, admin rotaları için WCAG 2.1 A/AA ve CSRF deny matrisi, davet,
backup, IP ban, editor/viewer hesap akışlarıyla aynı gerçek SQLite Edition
server fixture'ında çalışır.

Bu doğrulama `admin.content.write` override'ının etkin route davranışını ve
session iptalini kapsar. Her permission catalog entry için her rol ve her
mutation varyantının tam matrisi henüz planın Faz 4 açık kapsamındadır.

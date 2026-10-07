# Admin Studio ve müşteri akışları Chromium matrisi

**Tarih:** 2026-10-07  
**Sonuç:** 64/64 Chromium testi geçti  
**Kapsam:** Admin ve editör deneyimi, generated/dynamic CRUD, MFA, erişilebilirlik,
responsive davranış ve public sayfalar.

## Çalıştırma ortamı

Playwright paketi, seed edilmiş izole SQLite Edition ve geçerli tek kullanımlık
admin storage state ile çalıştırıldı. Generated Admin smoke testleri
`HYDRO_E2E_BASE_URL` ve `HYDRO_E2E_STORAGE_STATE` değişkenlerini kullanır;
kimlik doğrulaması olmayan rastgele bir `localhost:8090` sunucusu tam Admin
matrisi için geçerli fixture değildir. `HYDRO_E2E_ADMIN_EMAIL` ve
`HYDRO_E2E_ADMIN_PASSWORD` yalnızca izole yerel test hesabı içindir.

İzole sunucunun hazırlığı:

```sh
go build -trimpath -buildvcs=false -o "$TMPDIR/hydro" ./cmd/hydro
export HYDRO_MEDIA_DIR="$TMPDIR/media"
export HYDRO_ENABLE_DEMO_FRONTEND=true
"$TMPDIR/hydro" migrate --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
"$TMPDIR/hydro" seed-demo --confirm-demo --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
printf '%s\n' "$HYDRO_E2E_ADMIN_PASSWORD" | "$TMPDIR/hydro" create-admin --email "$HYDRO_E2E_ADMIN_EMAIL" --password-stdin --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite"
"$TMPDIR/hydro" serve --dev --database-dialect sqlite --database-dsn "$TMPDIR/edition.sqlite" --address 127.0.0.1:PORT
```

Sunucu çalışırken, geçici `auth-state.json` üretildikten sonra:

```sh
cd e2e
export HYDRO_E2E_BASE_URL=http://127.0.0.1:PORT
export HYDRO_E2E_STORAGE_STATE="$TMPDIR/auth-state.json"
npm run setup-auth
npm run generate
npx playwright test --project=chromium --workers=1 --reporter=line
```

Go ve repository kapıları:

```sh
go test ./... -count=1 -p 1 -timeout=5m
go vet ./...
go run ./tools/actionguard
go run ./tools/moduleguard -root modules -allow modules/api -dependencies modules/dependencies.json
go run ./tools/migrationguard -root migrations
go run ./tools/importguard -root examples/client-edition
go run ./tools/importguard -root examples/news-edition
go run ./tools/importguard -root examples/construction-edition
```

## Sonuç

- İlk doğrudan koşu fixture hazırlanmadan yapıldığı için generated Admin smoke
  testleri login sayfasına yönlendi. Bu, ürün hatası değil test ortamı
  eksiğiydi. Testler açıkça uygun admin session gerektirir.
- Doğru isolated server ve `setup-auth.mjs` ile tam Chromium matrisi **63/63**
  geçti. Kapsamda 15 viewport/responsive kontrolü, keyboard/focus ve adlandırma,
  admin role/CSRF, davet/parola/MFA, schema-driven dynamic collection CRUD,
  generated CRUD, rich-text/relation/media, taslak/yayın, kritik müşteri
  yolculukları ve manifestten üretilmiş public/Admin smoke testleri bulunur.
- İlk tam koşuda generated Admin axe taraması tek bir kontrast uyarısı verdi;
  izole tekil tekrar ve fixture-auth ile yapılan tam tekrar geçti. E2E helper'ı
  bundan sonra axe node failure summary'sini de assertion çıktısına ekler.
  Bu generated CRUD sayfası için sekiz tema/mod birleşimi de ayrıca axe-core
  taramasından geçirildi: HydroGo, shadcn, Nova ve Aurora için light/dark;
  **8/8 geçti.** Denetim, karanlık HydroGo modunda avatar kontrastını ve
  shadcn/Nova temalarının sabit avatar paletini ezmesini buldu. Palet renkleri
  tema değişimlerinden bağımsız hale getirildi; matriste tam Axe taraması artık
  tüm sekiz birleşimde koşuyor. Diğer Admin sayfaları ile tüm public/generated
  türler için tema bazlı WCAG kabulü hâlâ açık.
- Takipte generated CRUD liste görünümü sekiz aile/mod kombinasyonunda ayrı
  axe-core taramasından geçirildi: HydroGo, shadcn, Nova ve Aurora için light ve
  dark. **8/8 geçti.** Denetim, koyu modda HydroGo accent rengi üzerinde beyaz
  avatar metni kontrastını ve shadcn/Nova temalarının avatar paletini ezmesini
  buldu. Avatar palette renkleri temadan bağımsız hale getirildi; test bundan
  sonra her tema/modu doğrular. Bu alt matriste bütün Admin rotalarının veya
  public/generated sayfa türlerinin her temada tarandığı iddia edilmez.
- 2026-10-07 follow-up'ta generated CRUD create formu da aynı dört tema ailesi
  ve açık/koyu mod birleşimlerinde axe-core taramasından geçirildi: **8/8
  geçti**. Form kontrolü responsive Chromium akışında da yer alır; diğer Admin
  rotaları ve public sayfaların tüm tema/mod birleşimleri hâlâ kapsam dışıdır.
- Güvenlik takibinde editor/viewer oturumlarıyla medya liste erişimi denendi;
  yönetici `media.manage` iznini editörden kaldırınca yeni oturum `/admin/media`
  rotasından 403 alıyor, izin geri alındığında eski oturum iptal ediliyor.
  Tekil Chromium rol akışı **1/1** geçti. Medya handler'ları artık aynı
  `auth.Can` kontrolünü kullanır; izin reddi sunucu bileşeni eksikliğinden ayrı
  olarak 403 döndürülür.
- Admin hızlı araması artık pages/news, media, members ve site settings
  sonuçlarını sırasıyla `content.read`, `media.manage`, `members.manage` ve
  `site.manage` ile sınırlar. Gerçek editor oturumunda `content.read` deny
  override'ı sonrası sayfa/haber arama sonuçları yok; aynı admin-management
  Chromium akışındaki iki rol/izin testi geçti. Kaynak bazlı tam Go role ve
  override tablosu `internal/search/admin_http_test.go` içindedir.

2026-10-07 takip koşusunda `admin-management.spec.ts`, mevcut admin storage
state'i yeniden kullanarak 15 yönetim rotasının hem 390px hem 768px viewport'ta
200 döndüğünü ve yatay taşmadığını doğrulayan bir test ekledi. Böylece tam
Chromium matrisi **64/64** geçti (2.9 dakika). Admin desktop visual golden
koşusu 7/7 geçti. Tablet/mobile visual projeleri bu makinede WebKit bağımlı
tarayıcı başlatma kütüphaneleri eksik olduğu için açılamadı; aynı viewport
boyutlarındaki responsive kabul Chromium ile çalıştı. Bu sonuç fiziksel cihaz
veya WebKit görsel parity testi olarak değerlendirilmez.

Admin ve MFA grubu ayrıca `custom/admin-management.spec.ts`,
`custom/admin-password-reset.spec.ts` ve `custom/admin-mfa-settings.spec.ts`
ile yeni responsive kabul testi dahil 6/6 geçti. Suite aynı gerçek
login/challenge ve mutasyon akışlarını bağımsız tekrarlar.

Mobil kullanılabilirlik takibinde Admin Studio menüsü ve sık kullanılan
form/kart/aksiyon düğmeleri için 390px ve 768px genişliklerde en az 44px hedef
yüksekliği; bilgi ipuçları için en az 24×24px hedef boyutu tanımlandı. Rich-text
araç çubuğu kontrolleri yoğun düzeni korumak için bu düğme kuralının dışında.
`admin-management.spec.ts` 15 yönetim rotasını iki viewport'ta dolaşıp hedef
boyutlarını ölçüyor; test tek başına **1/1**, aynı dosyanın tüm yönetim/rol/
davet akışları **4/4**, `go test ./internal/ui -count=1` geçti. Bu, masaüstü
golden veya WebKit/gerçek cihaz doğrulamasının yerine geçmez.

Aynı mobil test hamburger menüsünü 390px'de açıp `aria-expanded` ve açık
durumunu kontrol eder; Escape ile kapatmayı, hemen yeniden açmayı ve bir menü
bağlantısı seçince hedef rotaya gidip menünün kapanmasını doğrular. Hemen
yeniden açma denemesi mevcut 250ms tıklama kilidinin kullanıcı girdisini
yuttuğunu ortaya çıkardı; redundant kilit kaldırıldı. Yeni menü akışıyla test
tek başına **1/1**, tam admin-management dosyası **4/4** geçti.

Faz 1 devamında Project Edition'e yeni bir `milestone` Blueprint kaynağı
eklendi. Aynı generated Admin registration helper'ı bu kaynağı ayrıca ID
tanımlamadan menüye aldı; API route ve onaylı migration ise Edition tarafından
açıkça bağlandı. Tarayıcı testi yeni koleksiyonda create/edit/delete ve API
readback yaptı, liste ve formu axe ile taradı ve 390px form taşmasını kontrol
etti. Koyu/açık dört tema ailesi matrisi mevcut Project listesinde tekrar geçti.
İlk yeni kaynak koşusunda koleksiyon açıklamasının kontrastı 3.7:1 çıktı;
paylaşılan `.dashboard-card-head .muted` stili semantik
`--ink-secondary` tokenına alındı ve Chromium matrisinin tamamı yeniden geçti.
Kaynak ve DB kontrat kanıtı `blueprint-derived-admin-activation-2026-10-07.md`
dosyasındadır.

## Follow-up — tüm Admin rotalarında tema WCAG matrisi

2026-10-07'de gerçek tek Edition admin oturumuyla 14 yönetim rotası, dört
Studio teması (HydroGo, shadcn, Nova, Aurora) ve açık/koyu modda axe-core
WCAG 2.1 A/AA taramasından geçirildi: **112/112 kombinasyon geçti**. İlk
koşuların yakaladığı düşük kontrastlı durum/izin rozetleri, medya metinleri ve
ikincil metin tonları düzeltildi. Test komutu ve kesin kapsam:
`docs/validation/admin-theme-accessibility-2026-10-07.md`.
Bu sonuç WebKit/cihaz görsel testi veya public/generated tüm türlerin tema
kapsaması olarak yorumlanmamalıdır.

## Linux WebKit tablet/mobile visual goldens

2026-10-07 takip doğrulamasında, `@playwright/test` 1.63.0 ile eşleşen resmi
`mcr.microsoft.com/playwright:v1.63.0-noble` container'ında Playwright'ın
WebKit varsayılanını kullanan `visual-tablet` ve `visual-mobile` projeleri
çalıştırıldı. İzole seeded SQLite Edition, geçici admin hesabı ve corporate
tema kullanıldı. Home, news, admin login, dashboard, page editor, news editor
ve media library için **14/14** Linux full-page screenshot baseline üretildi;
aynı session/fixture ile `--update-snapshots` olmadan tekrar çalıştırma da
**14/14** geçti. Canlı 390×664 WebKit viewport'unda news editor'ın sabit
kaydet aksiyonu ekran altında 44px ölçüldü. Bu golden seti yedi yüzeyi kapsar;
tüm yönetim rotaları, generated koleksiyonların tüm field varyantları veya
fiziksel iOS cihazı kapsamı iddia edilmez.

Kapsamı aynı koşullarda yeniden çalıştırmak için:

```sh
HYDRO_VISUAL=1 npx playwright test visual.spec.ts --project=visual-tablet --project=visual-mobile --workers=1
```

# Yerel geliştirme ortamı kurulumu (insan rehberi)

Bu rehber, HydroGo'nun kaynak kodunu (bu repo, `C:\dev\cms`) kendi
bilgisayarında çalıştırıp geliştirme yapmak için. Zaten oluşturulmuş bir
müşteri sitesini (Client Edition) geliştiriyorsan süreç biraz farklı ve
daha basit — ona `hydro project new` ile oluşturulan projenin kendi
`hydro run` komutu yeterli.

## Sunucuyu başlat

```sh
go run ./cmd/hydro run
```

Bu komut arka planda bir sürü şey yapıyor, hepsini senin için:

- **`http://127.0.0.1:8090`** adresinde bir "dev proxy" açıyor — tarayıcıda
  buraya gideceksin (8091 değil, o iç kullanım için).
- Veritabanı olarak varsayılan **SQLite** kullanıyor (`data/demo.sqlite`
  dosyası) — ayrı bir veritabanı kurmana gerek yok.
- Veritabanı migration'larını **otomatik** çalıştırıyor.
- `.templ` dosyaları varsa (admin paneli bileşenleri), onları otomatik
  derleyen bir "watcher" başlatıyor — elle bir derleme komutu çalıştırman
  gerekmiyor, dosyayı kaydet yeter.
- Kod değiştirdiğinde **otomatik yeniden derleyip sunucuyu yeniden
  başlatıyor** — derleme sürerken tarayıcıda "Updating Server" diye şirin
  bir bekleme ekranı görürsün, hata verip çökmez.
- Varsayılan olarak **örnek/demo içerik** (haberler vb.) oluşturuyor. Bu
  demo veri sadece bir "member" (üye) hesabı içeriyor, **admin hesabı
  içermiyor** — admin paneline girmek için ayrı bir adım gerekiyor,
  `admin-panel-access` rehberine bak.

## Bir değişiklik yaptıktan sonra

1. Dosyayı kaydet.
2. `hydro run` çalışıyorsa, otomatik olarak algılayıp yeniden derler —
   terminal çıktısında hata olup olmadığını izle.
3. Tarayıcıyı yenile, değişikliği gör.

## Proje içinde nerede ne var

- `cmd/hydro/` — komut satırı aracının kendisi (`serve`, `run`, `migrate`,
  `create-admin` gibi tüm komutlar burada).
- `internal/` — motorun kendi iç kodu.
- `modules/` — isteğe bağlı özellikler (sayfalar, haberler, üyeler, medya
  vb.), her biri kendi klasöründe.
- `themes/` — herkese açık sitenin görünümü (CSS/HTML), ayrı bir rehberi
  var: `frontend-development`.
- `migrations/` — veritabanı şema değişiklikleri, her biri kendi
  klasöründe, SQLite/PostgreSQL/MySQL için ayrı SQL dosyalarıyla.
- `docs/architecture/` — mimari kararların gerekçeleri (ADR'ler). Davranış
  değiştiren bir şey yapmadan önce ilgili olanı okumak proje kuralı
  (`AGENTS.md`'de yazıyor).

## Sık yapılan hatalar

- **`_templ.go` dosyasını elle düzenlemek** — bu dosya otomatik
  üretiliyor, bir sonraki derlemede senin değişikliğin silinir. Her zaman
  `.templ` uzantılı kaynak dosyayı düzenle.
- Küçük bir değişiklik için tüm test takımını (`go test ./...`) çalıştırıp
  dakikalarca beklemek — iterasyon sırasında sadece ilgili paketi test et,
  işin bitince tam takımı çalıştır.

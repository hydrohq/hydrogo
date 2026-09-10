# ADR-0022: Rıza temelli ürün analitiği ve yönetilebilir hukuki içerik

- **Durum:** Önerildi
- **Tarih:** 2026-09-10
- **Değiştirdiği ADR:** Yok
- **Yerine geçen ADR:** Yok

## Bağlam

Admin Studio'nun ziyaret, oturum süresi, ülke ve içerik etkileşimi raporları
sunabilmesi istenmektedir. Bu veriler bir kullanıcıyı doğrudan tanımlamak için
kullanılmasa bile tarayıcı davranışı ve çevrimiçi tanımlayıcılarla birlikte
kişisel veri riski taşıyabilir. Raporlama, kullanıcı erişiminin şartı haline
gelmemeli; zorunlu teknik çerezler ile analitik rızası aynı şey gibi
sunulmamalıdır.

Veri sorumlusu; aydınlatma metnindeki kimlik, amaç, hukuki sebep, aktarım,
saklama ve başvuru bilgilerini kendi faaliyetine göre doğrulamak zorundadır.
Core bu bilgileri uyduran veya hukuki uygunluğu garanti eden bir metin
üretmemelidir.

## Karar

- Analitik ayrı ve isteğe bağlı `analytics` modülüdür. Etkin değilse migration,
  route, istemci asset'i veya worker yüklemez.
- Modül varsayılan olarak kapalıdır. Etkinleştirildiğinde dahi analitik olay,
  ziyaretçi açıkça analitik tercihini vermeden kabul edilmez.
- Rıza kaydı amaç sürümü, tercih zamanı ve geri alma zamanı ile tutulur;
  zorunlu oturum/CSRF çerezinden ayrıdır. Ret veya geri alma sonrası yeni
  analitik event yazılmaz.
- İlk sürüm yalnız günlük agregalar üretir: onaylı ziyaret, oturum süresi
  toplamı, sayfa görünümü ve ülke kodu. IP adresi, tam URL query string'i,
  kullanıcı hesabı, kalıcı cihaz parmak izi ve reklam kimliği saklanmaz.
- Frontend, Core'a yalnız açık sözleşmeli consent ve event endpointleri
  üzerinden bağlanır. Ayrı frontend deployment'ı için istemci SDK/adapter
  açıkça rıza akışını uygular; Core session çerezleri takip amacıyla paylaşılmaz.
- Dashboard ve Raporlar yalnız analitik modül etkin ve yeterli sayıda onaylı
  aggregate bulunduğunda gerçek metrik gösterir; aksi durumda açıkça
  “veri yok / analitik kapalı” durumunu gösterir. Demo sayıları uydurulmaz.
- Hukuki metinler (gizlilik politikası, çerez politikası, KVKK aydınlatma
  metni ve analitik rıza metni) versiyonlu Pages kayıtları olarak yönetilir.
  Admin Studio, metinlerin eksik zorunlu alanlarını ve yayın durumunu gösterir;
  hukuki onay yerine geçmez.
- Rıza talebi, aydınlatma metninden ayrı sunulur; reddetmek hizmete erişimi
  engellemez. Her yeni analitik amaç/metin sürümü yeni tercih gerektirir.

## Alternatifler

- **Varsayılan açık, anonim üçüncü taraf analitik:** Rıza sınırını belirsiz
  kılar, veri kontrolünü zayıflatır ve self-hosted ürün hedefiyle çelişir.
- **Salt sunucu HTTP loglarından rapor üretmek:** Rıza yokken ziyaretçi
  davranışı ölçümüne dönüşme riski taşır; logların operasyon amacı ile ürün
  analitiği amacını karıştırır.
- **Yalnız hukuki metin linkleri:** Metin yayınını sağlar fakat rıza durumu,
  sürüm ve rapor erişimi arasında denetlenebilir bağlantı sağlamaz.

## Sonuçlar

### Kazanımlar

- Ürün metrikleri, açık rıza ve veri minimizasyonu sınırında üretilir.
- Dashboard ile Raporlar gerçek veri yokken bunu dürüstçe gösterir.
- Hukuki metinler içerik ekibi tarafından sürümlü olarak yönetilir.

### Maliyetler ve riskler

- Frontend starter'larının rıza banner'ını ve event sözleşmesini entegre
  etmesi gerekir.
- Ülke çözümleme için güvenilen bir edge/header sözleşmesi gerekir; ham IP
  depolanmaz ve header güvenilir değilse ülke “bilinmiyor” olur.
- Metin şablonları hukuk danışmanlığının yerine geçmez.

## Ölçek etkisi

- **Compact:** Günlük aggregate upsert'leri bounded tutulur; ham event akışı
  saklanmaz ve modül kapalıysa hiç kaynak ayırmaz.
- **Standard:** Gerekirse bounded, kalıcı aggregate işi ile raporlar
  yenilenir.
- **Distributed:** Rıza/event idempotency anahtarı, güvenilen proxy ülke
  çözümlemesi ve aggregate partition/retention sözleşmesi sertifikalanır.

## Veritabanı etkisi

- **SQLite:** Rıza kaydı ve günlük aggregate tabloları kısa transaction ile
  yazılır; sadece analytics modülü etkinse migration uygulanır.
- **PostgreSQL:** Aynı repository sözleşmesi ve gün bazlı unique aggregate
  upsert davranışı doğrulanır.
- **Gelecekteki adaptörler:** Tarih, ülke ve ölçüm boyutu unique kısıtları ile
  upsert/retention contract testini geçmelidir.

## Cache etkisi

Rapor cevapları kısa TTL ile cache'lenebilir; yeni aggregate veya rıza amaç
sürümü değiştiğinde ilgili dashboard/report key'leri geçersiz kılınır. Cache
arıza durumunda veritabanından bounded aggregate okunur; kişiye ait rıza
kaydı shared cache'e girmez.

## Deployment ve uyumluluk

Migration expand-only'dir. Eski binary analytics tablolarını kullanmaz; yeni
binary modül kapalıysa route/asset/worker açmaz. Amaç/metin sürümü değiştiğinde
eski consent yeni analitik event için geçerli sayılmaz.

## Doğrulama

- Rıza yokken, ret sonrası ve geri alma sonrası event endpointi kayıt yazmaz.
- Yalnız rıza sonrası aggregate artar; IP/query/user ID saklanmadığı testte
  doğrulanır.
- SQLite ve PostgreSQL aggregate upsert/retention contract testlerini geçer.
- Dashboard ve Raporlar kapalı/boş/etkin durumlarını doğru açıklar.
- Hukuki metin alanı yayınlanmamış veya eksikse rıza banner entegrasyonu
  analytics'i başlatmaz.

## Kaynaklar

- [KVKK — Çerez Uygulamaları Hakkında Rehber](https://www.kvkk.gov.tr/Icerik/7353/Cerez-Uygulamalari-Hakkinda-Rehber)
- [KVKK — Aydınlatma Yükümlülüğü](https://www.kvkk.gov.tr/Icerik/2033/Aydinlatma-Yukumlulugu-)
- [KVKK — Aydınlatma yükümlülüğü usul ve esasları](https://www.kvkk.gov.tr/Icerik/4132/aydinlatma-yukumlulugunun-yerine-getirilmesinde-uyulacak-usul-ve-esaslar-hakkinda-teblig)
- [ADR-0021: Headless browser session boundary](0021-headless-browser-session-boundary.md)

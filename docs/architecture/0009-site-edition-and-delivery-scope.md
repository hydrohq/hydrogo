# ADR-0009: İlk ürün, geliştirme sırası ve teslim sınırı

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05
- **İlişkili kararlar:** [ADR-0001](0001-product-and-deployment-model.md), [ADR-0006](0006-extensible-database-adapters.md)
- **Uygulama durumu:** Planlandı; çalışan ürün veya sertifikalı adapter henüz yok.

## Bağlam

Ana plan haber akışını ilk ürün gibi anlatırken backlog genel Site Edition öneriyordu. Ticari ürünün ilk kapsamı ile mimariyi doğrulayan ikinci modül birbirinden ayrılmalıdır. Tenancy kararı zaten ADR-0001'de kabul edilmiştir; yeniden açık soru değildir.

## Karar

İlk satılabilir ürün **Site Edition** olur. İlk hedef müşteri; kurumsal sayfalar, hizmetler, ekip/referans içerikleri ve iletişim formu isteyen, kendine ait tasarımla teslim edilen ajans müşterisidir. News ikinci doğrulama modülüdür; commerce ayrı ürün yatırımıdır.

| Teslim | Kapsam | Çıkış koşulu |
|---|---|---|
| İlk çalışan dilim | Auth, Pages, sınırlı Media, taslak/yayın, admin, public HTML ve public JSON | Aynı kullanım iki DB'de çalışır; yetkisiz erişim reddedilir |
| Pilot Site Edition | Navigation, settings, SEO/redirects, revisions, elle tanımlanan sınırlı iletişim formu/inbox, backup/restore, müşteri teması | Gerçek müşteri işi core fork olmadan tamamlanır |
| SDK doğrulaması | News: inceleme, zamanlama, özel yetkiler; ikinci Client Edition | İki farklı modül ve iki tema aynı extension sınırını kullanır |
| Kararlı v1 | Sertifikalı SQLite/PostgreSQL, belgeli upgrade, destek matrisi, güvenlik/restore testleri, doğrulanmış CRUD üretici | Hazırlık belgesindeki release kapıları geçilir |

Form inbox keyfi form tasarımcısı değildir. İlk form alanları edition kodunda tanımlanır; boyut sınırı, spam/rate limit, saklama ve kişisel veri erişimi bulunur. İleri Form Builder ertelenir.

Müşteri başına ayrı uygulama, veri, medya, sırlar ve yedek kullanılır. Çok kiracılı SaaS v1 kapsamında değildir. Bir deployment'taki locale'ler ayrı tenant değildir. Edition kimliği cache ve operasyon namespace'i sağlar; tek başına güvenlik izolasyonu sayılmaz.

## Şema ve özel kod

- Pages ve News önce elle yazılır. Ortak kalıp kanıtlandıktan sonra build sırasında kullanılan sürümlü collection tanımı standart CRUD üretir.
- Üretici tipli Go kodu, admin/API iskeleti, repository ve **incelenecek** dialect migration taslağı üretir. Çalışma anında şema tanımından otomatik DDL çalıştırılmaz.
- İş akışı, kayıt/alan yetkisi ve destructive değişiklikler tahmin edilmez. Üretilen erişim başlangıçta reddedilir; izinler açıkça tanımlanır.
- Yeni alan için belirlenmiş indeks, null/default, doğrulama ve export davranışı gerekir. Yeniden adlandırma drop+add olarak tahmin edilmez; açık alan kimliği ve rename kaydı gerekir.
- Müşterinin editörde içerik değiştirmesi build gerektirmez. Yeni collection şeması veya Go/Templ özelleştirmesi build ve gerekirse migration gerektirir. Bu sınır satış ve geliştirici belgelerinde belirtilir.

## Dağıtım kapsamı

Compact tek sunucu ilk pilot hedefidir. Standard/PostgreSQL aynı davranışlarla v1'e hazırlanır. Distributed mimarisi korunur ancak çok düğümlü operasyon, S3 uyumlu medya ve arıza testleri geçmeden sertifikalı seçenek olarak satılmaz. İlk pilotun bitmesi Distributed desteğinin tamamlandığı anlamına gelmez.

İlk üretim hedefi Linux amd64; Windows geliştirme akışı zorunlu doğrulanır. Linux arm64, Windows üretim ve diğer hedefler ayrı build/driver/işletim testlerinden sonra destek matrisine alınır. Tek binary sözü işletim sistemi ve mimari başına üretilen uygulama dosyasını anlatır; proxy, veri veya harici servislerin binary içinde olduğu anlamına gelmez.

## Alternatifler ve maliyet

Haber odaklı ilk ürün daha dar bir müşteriye hitap eder; genel Site Edition ajans tekrarını daha iyi ölçer. Her şeyi şemadan üretmek iş akışlarını zorla genelleştirir. İlk günden tüm topolojileri bitirmek pilotu geciktirir. Seçilen sıra, sonraki sertifikasyon işini ortadan kaldırmaz.

## Etki ve doğrulama

- SQLite/PostgreSQL contract testleri ilk Pages diliminden başlar; PostgreSQL sona bırakılmaz. Yeni DB'ler ADR-0006 sertifikasyonuna tabidir.
- Compact ek cache/queue servisi istemez; kaynaklar bounded olur. Distributed pool, worker ve backpressure doğrulaması kendi teslim kapısıdır.
- Pages public cache'i locale, publication ve edition sınırlarına uyar. Form/admin/preview shared cache'e girmez.
- Modül ekleme ve schema değişikliği release manifest'ine/migration planına girer; online olup olmadığı iki dialect için ayrı kanıtlanır.
- Aynı kapsamlı örnek işi mevcut geliştirme yöntemi ve HydroGo ile ölç: kurulum, tema, collection, özel kural, test, build/deploy ve hata düzeltme dahil. İlk tasarım emeği ile tekrarlanan kurulum süresini ayrı raporla. Tek başarılı örnekten 1–2 saat garantisi çıkarma.

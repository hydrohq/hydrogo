# ADR-0001: Ürün ve müşteri deployment modeli

- **Durum:** Kabul edildi
- **Tarih:** 2026-09-05

## Bağlam

PayloadCMS ve PocketBase tabanlı müşteri projelerinde çekirdeğin fork edilmesi veya doğrudan değiştirilmesi güncellemeleri zorlaştırdı. HydroGo hem pazarlanabilir bir ürün hem de ajansın tekrarlanan CMS geliştirmelerini hızlandıran bir platform olmalıdır.

## Karar

HydroGo, self-hosted ve geliştirici odaklı bir **CMS ürün fabrikası** olarak konumlandırılır.

- `HydroGo`: Müşteriye sunulan ürün ailesi ve deneyim.
- `HydroCore`: Sürümlü ve müşteri kodundan bağımsız motor.
- `Hydro Modules`: Pages, News, Media, Forms, SEO ve benzeri sürümlü yetenekler.
- `Hydro CLI`: Proje, şema, build, upgrade, backup ve deployment araçları.
- `Client Edition`: Marka, tema, açık modüller, içerik şemaları ve özel iş kuralları.

Her müşteri varsayılan olarak ayrı uygulama, veri tabanı, medya alanı, sırlar ve yedeklerle çalışır. Birden fazla düşük trafikli müşteri aynı makinede barınabilse de proses ve veri izolasyonu korunur.

Müşteri gereksinimleri HydroCore fork edilerek uygulanmaz. Yeni ihtiyaç:

1. Var olan modül veya sözleşmeli extension point ile çözülür.
2. Gerçekten genel bir yetenekse sürümlü Core/Module geliştirmesine dönüştürülür.
3. Müşteriye özgüyse `Client Edition` içinde kalır.

Bu yaklaşımın adı **Zero Core Fork** kuralıdır.

## Dağıtım biçimleri

- **Managed:** Ajans tarafından işletilen önerilen ticari model.
- **Self-hosted:** İmzalı release paketi ve işletim belgeleri müşteriye teslim edilir.
- **Container:** Kurumsal altyapılar için isteğe bağlı dağıtım biçimidir; ürünün zorunlu çalışma şekli değildir.

## Sonuçlar

- Core güncellemeleri merkezi olarak geliştirilebilir.
- Müşteri kodu küçük ve anlaşılır kalır.
- Her müşteri kendi Core ve modül sürümlerini sabitler.
- Çok sayıda müşteri için build matrisi, contract testleri ve fleet envanteri gerekir.
- Tam yapısal Templ override'ları müşteri başına yeniden build gerektirir; tema tokenları basit marka değişikliklerinde tercih edilir.

## Reddedilen seçenekler

- Müşteri başına HydroCore fork'u
- İlk sürümde multi-tenant SaaS
- Çalışma zamanında güvenilmeyen üçüncü taraf Go plugin yükleme
- Kubernetes'i varsayılan deployment yapmak

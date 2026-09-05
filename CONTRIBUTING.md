# Geliştirme akışı

Önce [AGENTS.md](AGENTS.md), manifesto, ana plan ve etkilenen ADR'leri oku. [Roadmap](docs/ROADMAP.md) içinden küçük bir kabul ölçütü seç. Mimari değişiklikte kabul edilmiş eski kararı sessizce değiştirme; yeni ADR ile ilişkilendir.

1. Kısa ömürlü çalışma branch'i aç; tek somut sonuç hedefle.
2. SDK'yı v0 tut; iki modül/Edition kanıtı olmadan genel sözleşmeyi dondurma.
3. Uygulamaya uygun testleri çalıştır. DB davranışı değişiyorsa SQLite ve PostgreSQL contract sonuçlarını kaydet.
4. Migration gerekiyorsa dialect ve rollback/uyumluluk metadata'sını birlikte hazırla; yayımlanmış migration'ı değiştirme.
5. İlgili roadmap maddesini yalnız kanıtla kapat; risk ve sınırlamaları PR açıklamasına yaz.

Go toolchain/driver/CI henüz seçilmedi. Gerçekte çalışmayan test veya build komutlarını başarılı diye raporlama. İlk kurulum R1 sırasında belgelenir. Sırlar, müşteri verileri, DB dosyaları, yedekler ve kişisel medya repoya girmez.

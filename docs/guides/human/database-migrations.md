# Veritabanı migration'ları (insan rehberi)

"Migration", veritabanı şemasında (tablolar, kolonlar) yapılan her
değişikliğin kayıt altına alınmış hali. HydroGo'da her migration kendi
klasöründe, **üç ayrı veritabanı türü için ayrı ayrı** SQL dosyası
içeriyor (SQLite, PostgreSQL, MySQL) — çünkü bu üç veritabanının SQL
söz dizimi birbirinden küçük ama kritik farklarla ayrılıyor.

## Migration nasıl çalıştırılır

```sh
hydro migrate --config config.yaml
```
Bu komut güvenli şekilde **tekrar tekrar çalıştırılabilir** — zaten
uygulanmış migration'ları otomatik atlıyor, sadece yenilerini uyguluyor.
Yerel geliştirmede (`hydro run`) bu zaten otomatik çalışıyor, elle
tetiklemen gerekmiyor.

## Bu oturumda yaşadığımız büyük ders: MySQL'in unutulması

Projenin kod tabanında, ~20 farklı dosyada **aynı hata kalıbı** bulundu:
SQL sorguları varsayılan olarak PostgreSQL söz dizimiyle yazılmış, SQLite
için ayrı bir durum eklenmiş, ama **MySQL hiç düşünülmemiş**. Sonuç: proje
MySQL ile çalıştırıldığında (tam da senin paylaşımlı hostingindeki gibi)
anlamsız hatalarla çöküyordu — "giriş yaparken 500 hatası" gibi.

Bunun pratik anlamı: eğer gelecekte "MySQL ile garip bir hata alıyorum ama
SQLite'ta her şey sorunsuz" diyen bir durumla karşılaşırsan, bu **bilinen
bir hata sınıfı** — kod muhtemelen MySQL'i unutmuş bir yerde. Bir ajana
"bu, daha önce bulduğumuz MySQL dialect hatası sınıfından olabilir mi,
kontrol et" dersen nereden başlayacağını bilir.

## Yeni bir migration eklerken (geliştiriciler için)

Her migration klasöründe `migration.yaml` adında bir metadata dosyası ve
her veritabanı türü için bir **checksum** (dosyanın "parmak izi") tutuluyor.
SQL dosyasını değiştirirsen, bu checksum'ı da güncellemen gerekiyor —
yoksa projenin otomatik kontrol araçları (`migrationguard`) bunu hata
olarak işaretler ve `git push` engellenir. Bir ajana migration
eklettiriyorsan bu adımı atlamaması gerektiğini bilir (kendi rehberinde
yazıyor), ama sen de bir migration'ın "bitti" sayılması için sadece SQL
dosyasının değil, checksum'ın da güncellenmiş olması gerektiğini bil.

## Migration'lar değiştirilemez

Bir migration bir kere uygulandıktan sonra, o migration dosyası bir daha
**değiştirilmez** — yeni bir ihtiyaç çıkarsa yeni bir migration eklenir.
Bu, mimari kararların (ADR) nasıl ele alındığına benziyor: eskiyi silmek
yerine üzerine yenisini eklemek.

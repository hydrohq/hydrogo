# Canlı siteyi güncelleme (insan rehberi)

Site ilk kez yayına alındıktan sonra, kod her değiştiğinde aynı döngüyü
tekrarlıyoruz: **birleştir → derle → yükle → doğrula → push et**. Bu
rehber, ilk kurulum değil, **zaten yayında olan** bir siteyi güncellemek
için.

## Adımlar

1. Değişikliğin bulunduğu branch'i ana branch'e (main) birleştir.
2. Binary'yi yeniden derle.
3. **Sadece binary mi değişti, yoksa başka bir şey de mi** (bir tema
   dosyası, `.htaccess`, bir config ayarı) kontrol et — kolayca atlanan bir
   adım, sadece binary yüklemek yetmeyebilir.
4. Dosyaları (FTP veya SSH ile) yükle.
   - **Paylaşımlı hosting (CGI):** yükledikten sonra başka bir şey
     yapmana gerek yok, bir sonraki ziyaretçi otomatik olarak yeni
     binary'yi kullanır.
   - **VPS:** yükledikten sonra servisi **yeniden başlatman** şart —
     yoksa eski binary bellekte çalışmaya devam eder, yeni dosya hiçbir
     işe yaramaz.
5. Veritabanı değişikliği varsa migration'ı çalıştır.
6. GitHub'a push et (git geçmişini canlıdaki durumla senkron tut).
7. **Siteyi gerçekten aç ve kontrol et** — sadece "yükleme başarılı"
   mesajına güvenme, push etmeden önce canlıda çalıştığını gözünle gör.

## Bilmen gereken küçük bir detay

Eğer art arda iki push yaparsan ("remote rejected... cannot lock ref" gibi
bir hata görebilirsin) — bu ciddi bir sorun değil, sadece iki push'un
çakışması, tekrar push etmen yeterli.

## En sık yapılan hata

Binary'yi günceleyip her şeyin bittiğini düşünmek, ama aslında o commit'te
bir tema dosyası veya config ayarı da değişmiş olup onu yüklemeyi unutmak.
Her deploy'da "bu commit'te binary dışında başka ne değişti?" sorusunu
sormak alışkanlık haline gelmeli.

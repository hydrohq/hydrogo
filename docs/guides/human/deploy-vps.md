# VPS'e deploy (insan rehberi)

Bu rehber, sana ait bir VPS'e (DigitalOcean, AWS EC2, Hetzner, vb. — SSH
erişimin olan herhangi bir sunucu) HydroGo'yu nasıl kalıcı, sürekli çalışan
bir servis olarak kuracağını anlatıyor. Paylaşımlı hosting'teki CGI
modelinden (her istekte yeni process) farklı olarak, burada HydroGo
**sürekli açık kalan tek bir process** olarak çalışıyor — bu, projenin asıl
tasarlandığı çalışma şekli, performans avantajları da buradan geliyor.

Bu süreci **gerçek bir VPS'te uçtan uca test ettim** — aşağıdaki her adım
çalıştığı doğrulanmış, varsayımsal değil.

## Başlamadan önce: VPS'te zaten ne var, kontrol et

VPS'in boş olduğunu varsayma. Üzerinde zaten çalışan başka bir uygulama
(Docker container'ları, başka bir site) olabilir. Özellikle **80 ve 443
portlarının zaten kullanımda olup olmadığını** kontrol ettirmek önemli —
eğer doluysa, mevcut uygulamayı durdurmadan farklı bir port kullanılmalı.

## Adımlar (özet)

1. **Binary'yi yükle** — Go'yu VPS'e kurmana gerek yok, binary'yi yerelde
   derleyip `scp` ile gönderiyoruz.
2. **`config.yaml`** hazırla — veritabanı (VPS'te SQLite de güvenle
   kullanılabilir, paylaşımlı hosting'deki kilitlenme riski burada yok),
   tema yolu, domain/IP.
3. **Migration çalıştır, admin hesabı oluştur** — VPS'te gerçek bir
   terminal (shell) olduğu için, paylaşımlı hosting'de yaptığımız
   "tek kullanımlık gizli dosya" numarasına gerek yok, komutları doğrudan
   çalıştırabiliyoruz.
4. **systemd servisi kur** — sunucu yeniden başlasa bile HydroGo'nun
   otomatik ayağa kalkmasını sağlıyor, çökerse otomatik yeniden başlatıyor.
5. **nginx reverse proxy kur** — dışarıdan gelen trafiği HydroGo'ya
   yönlendiriyor. **Çok önemli bir ayar var:** nginx'in gerçek domain/IP
   bilgisini HydroGo'ya doğru şekilde iletmesi gerekiyor, yoksa HydroGo
   isteği güvenlik gerekçesiyle reddediyor (bunu gerçek testte bizzat
   gördüm).
6. **SSL/HTTPS kur** — Let's Encrypt (`certbot`) ile ücretsiz sertifika,
   ya da önüne Cloudflare koyarsın (daha önceki paylaşımlı hosting
   rehberindeki gibi).

## Test ederken öğrendiğim önemli bir şey: bulut sağlayıcı güvenlik duvarı

VPS'te her şeyi doğru kurduğumda, **sunucunun kendisinden** test ettiğimde
her şey sorunsuz çalıştı. Ama **dışarıdan** (kendi bilgisayarımdan)
bağlanmaya çalıştığımda erişemedim. Sebebi: VPS'in işletim sistemi
seviyesindeki güvenlik duvarı (varsa) kapalı olsa bile, **bulut sağlayıcının
kendi panelinde ayrı bir güvenlik duvarı/Security Group ayarı** var — AWS,
GCP, DigitalOcean gibi her sağlayıcıda bu farklı isimlerle var ve
**SSH üzerinden düzeltilemez**, sağlayıcının web panelinden ilgili portu
(80, 443 vb.) açman gerekiyor.

Yani: "sunucuda her şey çalışıyor ama dışarıdan siteye giremiyorum"
durumuyla karşılaşırsan, önce bunu kontrol et.

## Test sonrası temizlik

Eğer bu bir deneme/doğrulama kurulumuysa (gerçek canlı site değilse), işin
bitince kurduğun her şeyi (systemd servisi, nginx ayarı, dosyalar) temizle
ve VPS'te önceden çalışan başka servislerin (varsa) hâlâ sorunsuz
çalıştığını doğrula.

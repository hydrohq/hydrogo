# Paylaşımlı hosting'e FTP ile deploy (insan rehberi)

Bu rehber, SSH ve Zamanlanmış Görevler (Scheduled Tasks) **olmayan** bir
Plesk (veya benzeri) paylaşımlı hosting hesabına HydroGo'yu CGI modunda
nasıl yayına alacağını, baştan sona, gerçek bir dağıtımda karşılaşılan
sorunlarla birlikte anlatıyor. Bir AI ajanından (Claude Code gibi) bunu
senin için yapmasını istiyorsan, ona bu dosyanın ikizi olan
`docs/guides/ai/deploy-ftp-shared-hosting.md` dosyasını işaret et — orada
aynı süreç, çalıştırılabilir komutlarla var.

## Neden FTP ve CGI?

HydroGo normalde sürekli açık kalan bir sunucu process'i olarak tasarlanmış.
SSH'siz paylaşımlı hostingde böyle bir process'i kendi başına ayakta
tutamazsın, bu yüzden Apache'nin her isteği kendi başlatıp kapattığı **CGI
modu** kullanılıyor. Dosyaları yüklemek için de en temel, her hosting
panelinde bulunan yöntem: **FTP**.

## Başlamadan önce elinde olması gerekenler

- **FTP bilgileri:** host (genelde domain adının kendisi, bazen sunucu IP'si
  gerekebilir), kullanıcı adı, şifre. Plesk'te bunu genelde "Dosyalar" veya
  "Hesap" sekmesinde, "FTP erişimi" başlığı altında bulursun.
- **Veritabanı bilgileri:** Plesk → Databases'ten oluşturduğun DB adı,
  kullanıcı adı, şifre. SQLite yerine **MySQL** önerilir — CGI'nin
  "her istek yeni process" modeliyle SQLite'ın dosya kilidi iyi
  anlaşmıyor.
- Hosting planında **CGI desteğinin açık olduğunu** doğrula (Hosting
  Ayarları → "Web komut dosyası" — PHP'nin FastCGI ayarıyla karıştırma,
  ayrı bir şey).

## 1. FTP bağlantısını test et

FileZilla gibi bir FTP istemcisiyle bağlan, ya da bir ajana "FTP'yi dene"
dersen `curl --list-only` ile aynı şeyi komut satırından yapar. Bağlantı
başarılıysa karşına `httpdocs`, `logs`, `private` gibi klasörler çıkar — bu,
gerçek webspace kökünün neresi olduğunu da sana gösterir.

## 2. Klasör yapısını kur

```
<webspace kökü>/
  private/            ← web'den ERİŞİLEMEZ, binary ve şifreler burada
    hydro
    config.yaml
    themes/<tema>/
    media/
  httpdocs/            ← gerçek, web'e açık klasör
    hydro.cgi
    .htaccess
```

`private/` klasörü **`httpdocs`'un yanında, içinde değil** olmalı —
içindeyse `config.yaml`'daki veritabanı şifren herkese açık bir URL'de
görünür hale gelir.

## 3. Binary'yi derle

Bir geliştirici/ajan bunu senin için `go build` ile (Windows'tan bile Linux
hedefine) derler. Sen sadece sonuçta ortaya çıkan `hydro` dosyasını
alacaksın.

## 4. Dosyaları FTP ile yükle

`hydro` binary'sini `private/`'a, `hydro.cgi` ve `.htaccess`'i
`httpdocs/`'a yükle. Yükledikten sonra `hydro` ve `hydro.cgi`'nin
**çalıştırılabilir (755)** izinlere sahip olduğunu kontrol et (FTP
istemcinde dosyaya sağ tık → İzinler/Permissions).

**Çok önemli bir tuzak:** `hydro.cgi` ve benzeri `.cgi` dosyalarını Plesk'in
kendi "Yeni Dosya" metin editörüne yapıştırarak **oluşturma**. Editörle
oluşturulan dosyalar farklı bir sahiplik alıyor ve sunucunun güvenlik
katmanı (suexec) bunu reddediyor — hata log'unda şöyle bir şey görürsün:
```
fopen: Permission denied
```
Çözüm: dosyayı yerelde hazırlayıp **upload** ile yükle, editöre yapıştırma.

`hydro.cgi` içeriği şöyle olmalı (yollar kendi webspace köküne göre
değişir):
```sh
#!/bin/sh
exec /webspace/kökü/private/hydro serve --cgi --config /webspace/kökü/private/config.yaml
```

## 5. `config.yaml`'ı hazırla

Veritabanı bilgilerini, tema adını ve domain'ini içeren bir `config.yaml`
dosyası gerekiyor. **YAML'de girintiye (boşluk sayısına) çok dikkat et** —
bir satırın başında fazladan boşluk olması tüm siteyi 500 hatasıyla
çökertebilir.

## 6. Migration ve admin hesabı oluştur

Bu en can alıcı kısım: SSH olmadan `hydro migrate` gibi komutları normalde
çalıştıramazsın. Çözüm, **tek kullanımlık, şifre korumalı küçük bir
tetikleyici dosya** oluşturup bir kez tarayıcıdan çağırmak, sonra hemen
silmek. Bir ajan bunu senin için hazırlayıp yükleyebilir — sen sadece
verdiği linki bir kez tarayıcından açıp "başarılı" çıktısını görüyorsun,
sonra o dosya siliniyor.

Aynı yöntemle bir admin kullanıcısı (e-posta + şifre) da oluşturuluyor.
**İlk girişten sonra bu şifreyi değiştirmen önerilir** — özellikle bir
AI asistanının ürettiği/bildiği bir şifreyse.

## 7. Test et

`https://senin-domainin.com/` adresini aç — sayfa açılmalı. `/login`'den
admin paneline gir, içerik/medya gibi temel ekranların 500 hatası vermeden
açıldığını doğrula.

## 8. Bir şeyler ters giderse: log'a bak

Plesk'te "Logs" sekmesi (veya FTP'den `logs/error_log` dosyası) genelde
tam neyin yanlış gittiğini söyler. HydroGo kendi hata mesajlarını da bu
log'a yazıyor, `stderr from .../hydro.cgi:` ile başlayan satırları ara.

## Sık karşılaşılan hatalar

| Belirti | Sebep | Çözüm |
|---|---|---|
| Ana sayfa 403 veriyor, hiç `hydro.cgi` çalışmamış | `.htaccess`'te `DirectoryIndex hydro.cgi` eksik, ya da hâlâ bir `index.html` duruyor | Satırı ekle / `index.html`'i sil |
| `fopen: Permission denied` | Dosya editörle oluşturulmuş | Upload ile yeniden yükle |
| "No such file or directory" | `hydro.cgi` içindeki yol yanlış | Gerçek webspace kökünü FTP'den doğrula |
| `decode config: yaml: ...` | `config.yaml` girinti hatası | Satır satır boşlukları kontrol et |
| `Table '...' doesn't exist` | Migration hiç çalışmamış | Adım 6'yı tekrarla |
| Site aniden DNS hatası veriyor | Nameserver değişikliği henüz dünya geneline yayılmadı | Birkaç saat bekle, `dnschecker.org`'dan kontrol et |

## Yapmaman gerekenler

- Tek kullanımlık migration/admin dosyalarını (adım 6) işin bitince
  silmeyi unutma — içlerinde şifre/token açık yazıyor.
- `config.yaml`, binary, ya da `media/` klasörünü asla `httpdocs/`'un
  içine koyma.
- SSH yoksa `hydro serve --fastcgi` kullanmaya çalışma — sürekli açık bir
  process gerektiriyor, SSH'sız ortamda ayakta tutamazsın.

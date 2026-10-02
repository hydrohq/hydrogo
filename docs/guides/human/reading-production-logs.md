# Canlı site log'larını okuma (insan rehberi)

Site'de bir şey ters giderse (500 hatası, beklenmedik davranış), gerçek
sebebi genelde **error log**'da yazar. Bu rehber, bu log'u nasıl bulup
okuyacağını anlatıyor.

## Log'a nasıl ulaşırsın

Plesk gibi panellerde genelde bir "Logs" / "Günlükler" sekmesi var —
oradan `error_log` dosyasını açabilirsin. Panel yavaşsa veya log çok
büyükse, bir ajana "FTP'den log'u çek" dersen aynı işi komut satırından
yapar.

## Log'da ne görüyorsun

HydroGo'nun kendi hata/istek kayıtları, Apache'nin genel satırlarının
**içine gömülü** geliyor — `AH01215: stderr from .../hydro.cgi:` yazan
satırların devamında, süslü parantezli (JSON) bir yapı göreceksin. Bu,
HydroGo'nun kendi log formatı — içinde hangi sayfaya gidildiği (`path`),
ne kadar sürdüğü (`duration_ms`), başarılı olup olmadığı (`status`) gibi
bilgiler var.

Eğer `AH01215:` sonrasında JSON değil de **düz bir hata metni** varsa
(örn. "No such file or directory"), bu genelde ciddi bir sorun —
uygulamanın normal çalışmaya bile başlayamadığı anlamına gelir (yanlış
dosya yolu, izin sorunu, bozuk config gibi).

## "Hiç log yok" her zaman kötü değil

Eğer bir sayfa cache'leniyorsa (sitenin hız için kullandığı mekanizma),
o sayfa cache'den servis edildiğinde **HydroGo hiç çalışmaz**, dolayısıyla
log'da hiçbir satır görünmez. Yani bir sayfaya defalarca gidip log'da
hiçbir şey görmemen, bir hata değil — tam tersine, cache'in çalıştığının
kanıtı olabilir.

## Pratik ipucu

Bir ajana "şu hatayı araştır" dersen, ona log'un tam metnini (kopyala-
yapıştır) vermen en hızlı çözümü bulmasını sağlar — ekran görüntüsü de
işe yarar ama metin olarak vermek (kopyalanabilir) daha güvenilir.

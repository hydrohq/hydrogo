# Domain'i Cloudflare'e bağlama ve SSL ayarları (insan rehberi)

## 1. Nameserver'ları değiştir

Domain'i satın aldığın yerde (registrar — GoDaddy, İsimtescil, Namecheap
vb.), nameserver'ları Cloudflare'in sana verdiği ikiliyle değiştir
(Cloudflare'de domain'i eklediğinde ekranda gösterir). Bu değişikliğin
**dünya geneline yayılması** (propagation) birkaç dakikadan 48 saate kadar
sürebilir — bu tamamen normal, acele etme.

**"Bende açılmıyor ama arkadaşımda açılıyor" gibi bir durumla
karşılaşırsan** — bu da normal, farklı internet sağlayıcılarının DNS
sunucuları farklı hızlarda güncelleniyor. `dnschecker.org` gibi bir
siteden dünya genelinde ne durumda olduğunu görebilirsin.

## 2. DNS kayıtlarını ekle

```
A     @          sunucu-ip     Proxied (turuncu bulut)
A     www        sunucu-ip     Proxied (turuncu bulut)
```
Mail ile ilgili kayıtlar (SPF, DKIM) ayrı bir konu —
`mail-smtp-setup.md` rehberine bak, onlar **asla** turuncu bulut olmamalı.

## 3. SSL modu

Cloudflare → SSL/TLS → Overview → **"Full (strict)"** seç. "Flexible"
seçme — ziyaretçi ile Cloudflare arası şifreli olsa da, Cloudflare ile
senin sunucun arası şifresiz kalır, bu da sitenin çerez/güvenlik
ayarlarıyla çakışıp yönlendirme döngüsüne (sürekli HTTPS'e atıp
döndürme) sokabilir.

Sunucunda henüz geçerli bir sertifika yoksa iki seçenek:
- Hosting panelinden ücretsiz **Let's Encrypt** sertifikası al.
- Cloudflare → SSL/TLS → Origin Server → **Create Certificate** ile
  Cloudflare'in kendi sertifikasını oluşturup hosting panelinden yükle.

Son olarak Cloudflare → SSL/TLS → Edge Certificates →
**"Always Use HTTPS"**'i aç.

## 4. Test et

Siteyi `https://` ile aç, tarayıcıda kilit simgesinin göründüğünü
doğrula. Bir şey garip görünürse, önce Cloudflare panelindeki domain
durumunun "Active" olduğunu kontrol et — "Pending Nameserver Update"
yazıyorsa henüz hazır değil demektir.

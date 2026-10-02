# Mail (SMTP) kurulumu — Resend örneğiyle (insan rehberi)

Sitenin şifre sıfırlama, bildirim gibi sistem maillerini güvenilir şekilde
göndermesi için Resend gibi bir "SMTP relay" servisi kullanıyoruz. HydroGo
kendi başına mail imzalamıyor, sadece standart bir SMTP istemcisi —
imzalama (DKIM) tamamen Resend'in (ya da hangi servisi kullanıyorsan onun)
sunucularında oluyor.

## 1. `config.yaml`'a eklenecek bilgiler

Resend panelinden alacağın API key ile:
```yaml
mail:
  host: "smtp.resend.com"
  port: 587
  from: "bilgi@senin-domainin.com"
  username: "resend"
  password: "re_xxxxxxxxxxxxxxxx"
  starttls: true
```
**Port 587 ve `starttls: true` kullan, 465'i değil** — bizim sistemde
doğrulanan kombinasyon bu.

**Dikkat:** `mail:` satırının başında yanlışlıkla fazladan boşluk
olmamalı, yoksa tüm `config.yaml` bozulur ve site çöker.

## 2. Resend'in istediği DNS kayıtlarını ekle

Resend panelinde domain'ini eklediğinde sana birkaç DNS kaydı verir
(genelde bir SPF, bir DKIM, bazen bir doğrulama kaydı). Bunları Cloudflare
(veya DNS sağlayıcın neyse) panelinden ekle.

**Çok önemli: DKIM kaydı asla "Proxied" (turuncu bulut) olmamalı**, her
zaman "DNS only" (gri bulut) olsun. Proxied olursa Resend'in imza
doğrulaması hiçbir zaman geçmez, mailler imzasız gider, spam'e düşer.

## 3. SPF — tek domain'de tek kayıt kuralı

Eğer domain'inin mailini aynı zamanda hosting sağlayıcın da (Plesk'in
kendi mail sunucusu gibi) kullanıyorsa, **iki ayrı SPF kaydı ekleme**.
Bir domain'de yalnızca **bir** SPF TXT kaydı olabilir — ikisini tek kayıtta
birleştirmen gerekiyor:
```
v=spf1 +a +mx +ip4:sunucu-ip include:resend-in-istedigi-deger ~all
```
İki ayrı SPF kaydı varsa, SPF tamamen geçersiz sayılır ve bu Gmail gibi
servisler için güçlü bir spam sinyalidir.

## 4. Mail hâlâ spam'e düşüyorsa, sırayla kontrol et

1. **Çift/çakışan SPF kaydı var mı?** (yukarıdaki madde)
2. **DKIM kaydı yanlışlıkla "Proxied" mi?**
3. **Eski, yanlış bir hesaptan kalma DKIM kaydı var mı?** Resend'de
   domain'i birden fazla kez (yanlış hesap, sonra doğru hesap gibi)
   eklediysen, aynı isimde iki farklı DKIM anahtarı kalmış olabilir —
   bu doğrulamayı tutarsızlaştırır. Hangisinin güncel olduğunu (genelde
   son eklenen) kontrol edip eskisini sil.
4. Resend panelinde domain **"Verified"** (doğrulanmış) görünüyor mu?
5. `from` adresi, doğruladığın domain ile aynı mı?

## 5. Düzelttikten sonra nasıl test edilir

Gerçek bir test maili gönder (örn. "şifremi unuttum" akışını tetikle),
Gmail'de mailin sağ üstündeki üç noktadan **"Orijinalini göster"**
seçeneğine bak — orada `SPF: PASS`, `DKIM: PASS`, `DMARC: PASS` yazıp
yazmadığını gör. Resend panelindeki "Verified" rozeti sadece DNS
kayıtlarının okunabilir olduğunu gösterir, gerçek bir mailin başarıyla
doğrulandığını değil — asıl kanıt bu "orijinalini göster" ekranı.

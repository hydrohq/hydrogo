# ADR-0013: Kimlik, yetki ve güvenlik sınırları

- **Durum:** Değiştirildi (kısmen) — restore silme geçmişinin kaynağı için [ADR-0014](0014-publication-routes-and-erasure-recovery.md); diğer güvenlik hükümleri geçerlidir.
- **Tarih:** 2026-09-05
- **İlişkili belgeler:** [Tehdit modeli](../security/THREAT_MODEL.md), [ADR-0010](0010-sdk-lifecycle-and-edition-manifest.md), [ADR-0011](0011-consistency-upgrade-and-recovery.md)
- **Uygulama durumu:** Yerel kullanıcı/session, rol, CSRF, MFA, reset, invite, limiter ve audit kontrolleri uygulanıp contract testleriyle doğrulandı. OIDC/SAML ve production provider/alerting kapsam dışı veya R3/R6 açık kapısıdır; güncel durum [roadmap](../ROADMAP.md) içindedir.

## Karar

İlk kimlik yöntemi yerel kullanıcı + parola + DB-backed opaque session olur. Kimlik sağlayıcısının çıktısı uygulama aktörüdür; authorization uygulama servisinde kalır. OIDC/SAML, mobil özel token ve API key dağıtımı kendi tehdit modeli/ADR'si olmadan eklenmez. Site Edition public JSON API'si kimlik isteyen mobil API desteği anlamına gelmez.

Kriptografik primitive, CSRF, parola hash'i ve TOTP algoritması sıfırdan yazılmaz; bakımı/lisansı/build uyumu değerlendirilen bağımlılık seçilir. Kütüphane sürümleri teknik spike sonunda sabitlenir. Mülkiyet ilkesi güvenlik algoritmalarını yeniden icat etme gerekçesi değildir.

## Yaşam döngüsü

- İlk yönetici işletim erişimi olan kişinin açık bootstrap komutuyla oluşturulur; herkese açık ilk-kullanıcı kayıt endpoint'i yoktur. Son kullanılabilir yönetici hesabını yanlışlıkla kilitleyen silme/yetki değişikliği engellenir.
- Session kimliği kriptografik rastgele, DB kaydı hash'li, cookie host-only/HttpOnly/Secure/SameSite=Lax olur. Login ve privilege değişiminde session yenilenir; idle/absolute süreler ayrı uygulanır. Üretimde HTTPS zorunludur; local geliştirme istisnası üretim modunda açılamaz.
- Kullanıcı disable, parola sıfırlama ve kritik rol değişiminde session'lar iptal edilir. Kayıt/alan politikaları her uygulama servisi çağrısında değerlendirilir; uzun ömürlü role cache'i yeni yetkiyi gizlemez.
- Davet ve reset token'ları ayrı purpose, kullanıcı bağı, kısa süre, hash saklama ve tek kullanım taşır. Tüketim ve parola değişikliği aynı transaction'dır. Yanıt hesap varlığını ifşa etmez; URL güvenilen config origin'inden üretilir.
- Yönetici/publisher için MFA, internetten erişilebilir ticari pilot kapısıdır. İlk yöntem TOTP ve hash'li tek kullanımlık recovery code; seed şifreli saklanır. Kayıt etkinleşmeden challenge doğrulanır; aynı zaman aralığındaki kod tekrar kullanımı test edilir. Recovery/disable yeni kimlik doğrulama ve audit ister.
- Kaybolan MFA için işletim yetkilisinin belgeli CLI kurtarma yolu vardır; genel web bypass yoktur. Recovery ve parola sıfırlama olayları kullanıcıya güvenli bildirim üretir.
- Password hashing, login ve MFA denemeleri ayrı concurrency/rate limit bütçesi taşır. Tek isteğin güvenli hash maliyeti toplu saldırıda sınırsız bellek kullanımına dönüşmez.

## Authorization ve ağ sınırı

Route sınıflandırması default-deny olur. Admin, public API, özel JSON endpoint ve background job aynı servis politikalarını kullanır. Role ek olarak kayıt sahipliği, durum, locale ve alan allowlist'i değerlendirilir. Hook'ların politikayı gevşetmesi desteklenmez.

Admin state-changing isteklerinde CSRF zorunludur; `SameSite` ek savunmadır. CORS varsayılan çapraz-origin erişimi açmaz. Proxy header'ları yalnız güvenilen proxy'den kabul edilir; allowed-host listesi ve canonical origin config'i vardır. Health/readiness public çıktıları sürüm/sır/DB ayrıntısı taşımaz; ayrıntılı diagnostics ve metrics korunur.

Outbound webhook/remote fetch başlangıçta genel kullanıma açılmaz. Ekleneceğinde izinli hedef, özel/link-local/metadata ağ engeli, DNS/redirect yeniden kontrolü, timeout ve yanıt boyutu şarttır. Yalnız URL'nin ilk string kontrolü yeterli kabul edilmez.

## Audit ve güven sınırının sınırı

Audit uygulama API'sinde append-only'dir. PostgreSQL runtime rolü için audit UPDATE/DELETE yetkisi kaldırılır; migration/bakım kimliği ayrıdır. SQLite dosyasına veya sunucuya tam erişimi olan kişiye karşı bu DB rol güvencesi yoktur. Ayrı OS kullanıcısı/dosya izinleri temel sınırdır. Kurcalama tespiti gereken müşteri için dış append-only/retention kontrollü hedef ayrıca sertifikalanır; yerel kayıt “root bile değiştiremez” diye sunulmaz.

Session/reset/preview token'ları, parola, TOTP seed ve form gövdeleri log/audit'e yazılmaz. Veri silme/saklama kuralları revisions, soft-delete, search, medya ve backup retention'ı birlikte kapsar; backup'tan restore silinmiş veriyi tekrar görünür yapmadan, restore edilen DB'den bağımsız dış silme günlüğüyle uzlaştırılır. Dayanıklı niyet, checkpoint, retention ve eksik günlükte izolasyon kuralları ADR-0014'tedir.

## Etkiler, alternatifler ve doğrulama

Stateless uzun ömürlü JWT yerine DB session hızlı iptal ve blue/green sürekliliği sağlar; DB'ye bağlı erişim maliyeti vardır. İlk aşamada genel identity framework yerine küçük adapter sınırı seçilir. MFA ilk geliştirme dilimini değil, ticari pilotu engeller.

SQLite/PostgreSQL'de tek kullanımlık token tüketimi, oturum iptali ve auth/audit transaction testleri aynıdır. Session shared response cache'e girmez. Compact auth concurrency bounded'dır; Distributed rate limit/oturum iptali çoklu instance testinden geçer. Secret key rotasyonunda eski/yeni slotun desteklediği key ID'leri kontrollü örtüşür; süresiz eski anahtar kabul edilmez. Restore güvenlik kuralları ADR-0011'dedir.

Güvenlik test senaryoları ve artık riskler [tehdit modelinde](../security/THREAT_MODEL.md) izlenir. Model bir penetrasyon testi veya güvenlik sertifikası değildir.

## Kaynaklar

- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html): Session yaşam döngüsü ve cookie savunmaları.
- [OWASP Forgot Password](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html): Tek kullanımlık kurtarma token'ı ve hesap keşfini sınırlama.
- [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html): Dış hedef ve ağ doğrulama sınırları.

Bu kaynaklar kontrollerin gerekçesidir; pilotta MFA zorunluluğu gibi teslim tercihleri HydroGo tasarım kararıdır.

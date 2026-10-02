# Admin paneline (Admin Studio) nasıl girilir (insan rehberi)

## Önemli: otomatik admin hesabı yok

Ne yerel geliştirme ortamı (`hydro run`) ne de yeni bir canlı kurulum,
otomatik olarak bir admin hesabı oluşturmaz. Demo veri (`seed-demo`)
**bilinçli olarak** sadece örnek bir üye (member) hesabı oluşturur, admin
hesabı değil — yani "acaba demo şifresi nedir" diye aramana gerek yok,
öyle bir şey yok, admin hesabını **sen oluşturacaksın**.

## Admin hesabı oluşturma

Bir ajana (Claude Code gibi) "bana admin hesabı aç" dersen şu komutu
senin için çalıştırır:
```sh
hydro create-admin --email <e-posta> --password-stdin --config <config.yaml>
```
Başarılı olursa `Created admin user: ...` yazısını görürsün.

SSH'in olmayan bir paylaşımlı hosting'teysen (bizim ardalabs.xyz'de
yaptığımız gibi), bu komutu doğrudan çalıştıramazsın — bunun yerine
`docs/guides/human/deploy-ftp-shared-hosting.md`'deki tek kullanımlık
tetikleyici dosya yöntemi kullanılıyor.

## Giriş yapma

- Yerel geliştirmede: `http://127.0.0.1:8090/login`
- Canlı sitede: `https://senin-domainin.com/login`

`/admin` adresine gidersen, giriş yapmamışsan otomatik olarak `/login`'e
yönlendirilirsin — ikisi de aynı kapıya çıkar.

## Roller

Üç rol var: **Admin**, **Editor**, **Viewer**. `create-admin` komutuyla
oluşturulan hesap her zaman Admin rolünde başlar. Başka bir kullanıcının
rolünü değiştirmek istersen, admin panelindeki kullanıcı yönetimi
ekranından yapabilirsin.

## İki faktörlü doğrulama (MFA)

Admin Studio, TOTP tabanlı (Google Authenticator tarzı) MFA destekliyor.
Varsayılan olarak zorunlu değil, panelden isteğe bağlı olarak açılıyor —
`config.yaml`'da belirli roller için zorunlu da yapılabilir.

**Yerel geliştirmede kendini kilitlersen** (MFA'yı test ederken şifreni
unutursan vb.) en hızlı çözüm: yeni bir e-posta ile tekrar
`create-admin` çalıştırmak, ya da zaten tek kullanımlık olan `data/demo.sqlite`
dosyasını silip sıfırdan başlamak — bu gerçek/canlı veri değil, demo veri.

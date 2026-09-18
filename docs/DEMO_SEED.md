# Demo seed kapsamı

`seed-demo --confirm-demo` yalnızca yerel geliştirme ve tarayıcı doğrulaması
içindir. Açık onay olmadan komut fail-closed olur; üretim verisi, gerçek müşteri
hesabı veya canlı ödeme bilgisi olarak kullanılmamalıdır.

Demo seed varsayılan runtime yüzeyi değildir. HydroGo'nun `/` adresi backend
runtime durumunu gösterir; demo frontend'i seed işleminden sonra
`/public/pages/{locale}/demo/html` adresinden açılır. Üretim kurulumlarında demo
seed çalıştırılmamalı ve ürün frontend'i `site.home_path` ile açıkça bağlanmalıdır.
Seed edilmiş demo sayfaları public page route'larından varsayılan olarak gizlidir.
Yerel tarayıcı doğrulaması gerektiğinde `HYDRO_ENABLE_DEMO_FRONTEND=true` ile
sunucuyu başlatmak gerekir.

Seed şu akışları görünür ve test edilebilir hale getirir:

- Sayfalar, haberler, kapak medya varlıkları ve Commerce ürün/varyant/stok kayıtları.
- Public frontend üzerinden kullanılabilen örnek üye hesabı: `member@hydrogo.local` / `memberpass123`.
- Üye hesabında gösterilmek üzere örnek sipariş ve sipariş kalemi.
- SQLite ve PostgreSQL için idempotent kayıtlar; mevcut aynı kimlikler ezilmez.

Örnek parola yalnızca yerel demo içindir. Dağıtım veya müşteri projesi oluşturulurken seed kapatılmalı ya da demo üye/parola mutlaka değiştirilmelidir. Canlı ödeme sağlayıcısı, gerçek e-posta, gerçek medya depolaması ve üretim analitiği seed tarafından taklit edilmez.

Üye hesabı Admin Studio hesabından ayrıdır. `/member/orders` yalnızca oturum açmış üyenin kendi siparişlerini döndürür; admin oturumu bu public üye akışına yetki vermez.

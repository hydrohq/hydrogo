# Demo seed kapsamı

`seed-demo` yalnızca yerel geliştirme ve tarayıcı doğrulaması içindir. Üretim verisi, gerçek müşteri hesabı veya canlı ödeme bilgisi olarak kullanılmamalıdır.

Seed şu akışları görünür ve test edilebilir hale getirir:

- Sayfalar, haberler, kapak medya varlıkları ve Commerce ürün/varyant/stok kayıtları.
- Public frontend üzerinden kullanılabilen örnek üye hesabı: `member@hydrogo.local` / `memberpass123`.
- Üye hesabında gösterilmek üzere örnek sipariş ve sipariş kalemi.
- SQLite ve PostgreSQL için idempotent kayıtlar; mevcut aynı kimlikler ezilmez.

Örnek parola yalnızca yerel demo içindir. Dağıtım veya müşteri projesi oluşturulurken seed kapatılmalı ya da demo üye/parola mutlaka değiştirilmelidir. Canlı ödeme sağlayıcısı, gerçek e-posta, gerçek medya depolaması ve üretim analitiği seed tarafından taklit edilmez.

Üye hesabı Admin Studio hesabından ayrıdır. `/member/orders` yalnızca oturum açmış üyenin kendi siparişlerini döndürür; admin oturumu bu public üye akışına yetki vermez.

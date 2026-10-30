# HydroGo Boutique Storefront Starter (`@hydrogo/boutique-starter`)

Dijital butikler (giyim, aksesuar, kozmetik, takı, butik tasarım atölyeleri) için **HydroGo** üzerinde çalışan, mobil öncelikli (Instagram bio trafiği odaklı) lüks vitrin arayüzü.

## Özellikler

- **Sıfır Yeniden Derleme ile Canlı Tema Motoru (`src/lib/theme.ts`)**: `/api/v1/content/store_settings` kaydındaki `theme_preset` (`minimal_cream`, `noir_gold`, `pastel_blush`, `emerald_silk`), `bg_primary_color`, `text_primary_color`, `accent_color`, `font_heading` ve `border_radius_style` değerlerini çalışma anında CSS değişkenlerine (`--bg-primary`, `--text-primary`, `--accent`, `--font-heading`, `--radius`) dönüştürür. Mağaza sahibi **AI Mağaza Asistanı (Copilot)** üzerinden *"Temayı koyu ve altın rengi yap"* dediğinde vitrin anında güncellenir.
- **WhatsApp Hızlı Sipariş (`src/lib/whatsapp.ts`)**: Müşteri ürün kartından beden ve renk seçip **"WhatsApp ile Hemen Sipariş Ver"** butonuna bastığında tüm ürün, beden, renk ve fiyat bilgisi hazır mesaj olarak mağazanın WhatsApp hattına (`wa.me`) yönlendirilir.
- **Kredi Kartı / Shopier / İyzico Bağlantısı**: Ürün kaydında `payment_link` tanımlıysa doğrudan ödeme butonu gösterilir.
- **60 Saniyede AI Mağaza Kurulumu**:
  ```bash
  go run ./tools/factory \
    --tenant ipek-butik \
    --name "İpek Butik" \
    --phone "+905551112233" \
    --prompt "Kadın ipek gömlek ve keten takım satan lüks butik"
  ```

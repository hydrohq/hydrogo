<script lang="ts">
  import { cart } from '$lib/stores/cart.svelte';

  let { open = $bindable(false), storePhone = '' }: { open: boolean; storePhone?: string } = $props();

  let buyerName = $state('');
  let buyerEmail = $state('');
  let buyerPhone = $state('');
  let buyerAddress = $state('');
  let buyerCity = $state('İstanbul');
  let note = $state('');
  let paymentMethod = $state<'shopier' | 'whatsapp'>('shopier');

  let loading = $state(false);
  let errorMessage = $state<string | null>(null);

  function close() {
    open = false;
    errorMessage = null;
  }

  async function handleSubmit(e: Event) {
    e.preventDefault();
    if (cart.items.length === 0) return;

    if (!buyerName.trim() || !buyerPhone.trim() || !buyerAddress.trim()) {
      errorMessage = 'Lütfen ad, telefon ve teslimat adresini eksiksiz doldurun.';
      return;
    }

    loading = true;
    errorMessage = null;

    try {
      if (paymentMethod === 'whatsapp') {
        const orderSummary = cart.items
          .map((i) => `• ${i.name} (${i.quantity} adet) - ${((i.priceCents * i.quantity) / 100).toFixed(2)} ${i.currency}`)
          .join('\n');
        const text = encodeURIComponent(
          `Merhaba, web sitenizden sipariş vermek istiyorum:\n\n${orderSummary}\n\nToplam: ${cart.formattedTotal}\n\nAlıcı: ${buyerName}\nTelefon: ${buyerPhone}\nAdres: ${buyerAddress}, ${buyerCity}\nNot: ${note}`
        );
        const cleanPhone = (storePhone || '905550000000').replace(/[^0-9]/g, '');
        window.open(`https://wa.me/${cleanPhone}?text=${text}`, '_blank');
        cart.clear();
        close();
        return;
      }

      // Shopier Checkout Call
      const res = await fetch('/api/v1/storefront/order', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          customer_name: buyerName,
          customer_email: buyerEmail || `${buyerPhone.replace(/[^0-9]/g, '')}@musteri.local`,
          customer_phone: buyerPhone,
          shipping_address: buyerAddress,
          city: buyerCity,
          items: cart.items.map((i) => ({
            id: i.id,
            name: i.name,
            price_cents: i.priceCents,
            quantity: i.quantity,
          })),
          note,
        }),
      });

      const json = await res.json();
      if (!res.ok) {
        throw new Error(json.message || 'Sipariş oluşturulamadı.');
      }

      // If server returned a Shopier payment form or redirect URL
      if (json.data?.shopier_form_html) {
        const div = document.createElement('div');
        div.innerHTML = json.data.shopier_form_html;
        document.body.appendChild(div);
        const form = div.querySelector('form');
        if (form) {
          cart.clear();
          form.submit();
          return;
        }
      } else if (json.data?.payment_url) {
        cart.clear();
        window.location.href = json.data.payment_url;
        return;
      }

      // Fallback success
      cart.clear();
      close();
      alert('Siparişiniz başarıyla alındı.');
    } catch (err: any) {
      errorMessage = err.message || 'Ödeme başlatılırken bir hata oluştu.';
    } finally {
      loading = false;
    }
  }
</script>

{#if open}
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-xs z-50 flex items-center justify-center p-4 overflow-y-auto"
    onclick={(e) => {
      if (e.target === e.currentTarget) close();
    }}
    onkeydown={(e) => e.key === 'Escape' && close()}
    role="dialog"
    aria-modal="true"
    tabindex="0"
  >
    <div class="bg-card border border-border rounded-xl shadow-2xl max-w-lg w-full overflow-hidden my-8">
      <!-- Modal Header -->
      <div class="px-6 py-4 border-b border-border flex items-center justify-between">
        <div>
          <h3 class="text-base font-semibold text-foreground">Hızlı ve Güvenli Sipariş</h3>
          <p class="text-xs text-muted-foreground mt-0.5">Sipariş bilgilerinizi tamamlayın</p>
        </div>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground p-1 rounded-md hover:bg-secondary transition-colors"
          onclick={close}
          aria-label="Kapat"
        >
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Form Body -->
      <form onsubmit={handleSubmit} class="p-6 space-y-4">
        {#if errorMessage}
          <div class="p-3 rounded-lg border border-destructive/30 bg-destructive/10 text-destructive text-xs">
            {errorMessage}
          </div>
        {/if}

        <!-- Order Summary Pill -->
        <div class="p-3 rounded-lg bg-secondary/50 border border-border flex items-center justify-between text-xs">
          <span class="text-muted-foreground">{cart.totalCount} Adet Ürün</span>
          <span class="font-bold text-foreground">{cart.formattedTotal}</span>
        </div>

        <!-- Contact Fields -->
        <div class="space-y-3">
          <div>
            <label for="buyer-name" class="block text-xs font-medium text-foreground mb-1">Ad Soyad *</label>
            <input
              id="buyer-name"
              type="text"
              required
              bind:value={buyerName}
              placeholder="Örn: Ahmet Yılmaz"
              class="w-full h-9 px-3 rounded-md border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label for="buyer-phone" class="block text-xs font-medium text-foreground mb-1">Telefon *</label>
              <input
                id="buyer-phone"
                type="tel"
                required
                bind:value={buyerPhone}
                placeholder="05XX XXX XX XX"
                class="w-full h-9 px-3 rounded-md border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
            <div>
              <label for="buyer-email" class="block text-xs font-medium text-foreground mb-1">E-Posta (İsteğe bağlı)</label>
              <input
                id="buyer-email"
                type="email"
                bind:value={buyerEmail}
                placeholder="ahmet@example.com"
                class="w-full h-9 px-3 rounded-md border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
          </div>

          <div>
            <label for="buyer-address" class="block text-xs font-medium text-foreground mb-1">Teslimat Adresi *</label>
            <textarea
              id="buyer-address"
              required
              rows="2"
              bind:value={buyerAddress}
              placeholder="Cadde, sokak, bina ve daire no..."
              class="w-full p-2.5 rounded-md border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            ></textarea>
          </div>

          <div class="grid grid-cols-2 gap-3">
            <div>
              <label for="buyer-city" class="block text-xs font-medium text-foreground mb-1">Şehir</label>
              <input
                id="buyer-city"
                type="text"
                bind:value={buyerCity}
                placeholder="İstanbul"
                class="w-full h-9 px-3 rounded-md border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
            <div>
              <label for="order-note" class="block text-xs font-medium text-foreground mb-1">Sipariş Notu</label>
              <input
                id="order-note"
                type="text"
                bind:value={note}
                placeholder="Zil çalmasın vb."
                class="w-full h-9 px-3 rounded-md border border-input bg-background text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
          </div>
        </div>

        <!-- Payment Method Selection -->
        <div class="space-y-2 pt-2 border-t border-border">
          <span class="block text-xs font-medium text-foreground">Ödeme Yöntemi</span>
          <div class="grid grid-cols-2 gap-2">
            <button
              type="button"
              class="p-3 rounded-lg border text-left transition-colors flex flex-col gap-1 {paymentMethod === 'shopier' ? 'border-primary bg-primary/10 text-foreground' : 'border-border bg-card text-muted-foreground hover:bg-secondary/50'}"
              onclick={() => (paymentMethod = 'shopier')}
            >
              <div class="flex items-center justify-between">
                <span class="text-xs font-semibold text-foreground">Kredi / Banka Kartı</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-500 font-mono">3D Secure</span>
              </div>
              <span class="text-[11px] text-muted-foreground">Shopier güvencesiyle anında</span>
            </button>

            <button
              type="button"
              class="p-3 rounded-lg border text-left transition-colors flex flex-col gap-1 {paymentMethod === 'whatsapp' ? 'border-emerald-500 bg-emerald-500/10 text-foreground' : 'border-border bg-card text-muted-foreground hover:bg-secondary/50'}"
              onclick={() => (paymentMethod = 'whatsapp')}
            >
              <div class="flex items-center justify-between">
                <span class="text-xs font-semibold text-foreground">WhatsApp Sipariş</span>
                <span class="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-500 font-mono">Doğrudan</span>
              </div>
              <span class="text-[11px] text-muted-foreground">Temsilci ile teyitli</span>
            </button>
          </div>
        </div>

        <!-- Submit Button -->
        <div class="pt-3">
          <button
            type="submit"
            disabled={loading}
            class="w-full h-11 bg-primary text-primary-foreground font-medium rounded-lg hover:bg-primary/90 transition-colors flex items-center justify-center gap-2 text-sm disabled:opacity-50"
          >
            {#if loading}
              <svg class="animate-spin w-4 h-4 text-primary-foreground" fill="none" viewBox="0 0 24 24">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"></path>
              </svg>
              <span>İşleniyor...</span>
            {:else}
              <span>{paymentMethod === 'shopier' ? 'Ödemeye İlerle' : 'WhatsApp ile Gönder'}</span>
              <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M14 5l7 7m0 0l-7 7m7-7H3" />
              </svg>
            {/if}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}

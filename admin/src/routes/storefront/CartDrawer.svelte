<script lang="ts">
  import { cart } from '$lib/stores/cart.svelte';

  let { oncheckout }: { oncheckout: () => void } = $props();

  function close() {
    cart.toggle(false);
  }
</script>

{#if cart.isOpen}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-xs z-50 transition-opacity"
    onclick={close}
    onkeydown={(e) => e.key === 'Escape' && close()}
    role="button"
    tabindex="0"
    aria-label="Kapat"
  ></div>

  <!-- Slide-over Drawer -->
  <div
    class="fixed inset-y-0 right-0 z-50 w-full max-w-md bg-card border-l border-border shadow-2xl flex flex-col transform transition-transform duration-200 ease-in-out"
    role="dialog"
    aria-modal="true"
    aria-label="Alışveriş Sepeti"
  >
    <!-- Header -->
    <div class="px-6 py-4 border-b border-border flex items-center justify-between">
      <div class="flex items-center gap-2">
        <h2 class="text-base font-semibold tracking-tight text-foreground">Alışveriş Sepeti</h2>
        <span class="text-xs px-2 py-0.5 rounded-full bg-secondary text-secondary-foreground font-mono">
          {cart.totalCount}
        </span>
      </div>
      <button
        type="button"
        class="text-muted-foreground hover:text-foreground p-1.5 rounded-md hover:bg-secondary transition-colors"
        onclick={close}
        aria-label="Kapat"
      >
        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Items List -->
    <div class="flex-1 overflow-y-auto p-6 space-y-4">
      {#if cart.items.length === 0}
        <div class="h-full flex flex-col items-center justify-center text-center py-12">
          <div class="w-12 h-12 rounded-full bg-secondary flex items-center justify-center text-muted-foreground mb-3">
            <svg class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
            </svg>
          </div>
          <p class="text-sm font-medium text-foreground">Sepetiniz henüz boş</p>
          <p class="text-xs text-muted-foreground mt-1">Koleksiyondan ürün seçerek alışverişe başlayın.</p>
        </div>
      {:else}
        {#each cart.items as item (item.id)}
          <div class="flex gap-4 p-3 rounded-lg border border-border bg-background/50">
            {#if item.image}
              <img
                src={item.image}
                alt={item.name}
                class="w-16 h-16 object-cover rounded-md border border-border shrink-0"
              />
            {:else}
              <div class="w-16 h-16 rounded-md bg-secondary border border-border flex items-center justify-center text-muted-foreground shrink-0">
                <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                </svg>
              </div>
            {/if}

            <div class="flex-1 min-w-0 flex flex-col justify-between">
              <div class="flex items-start justify-between gap-2">
                <h4 class="text-sm font-medium text-foreground truncate">{item.name}</h4>
                <button
                  type="button"
                  class="text-muted-foreground hover:text-destructive p-1 rounded transition-colors"
                  onclick={() => cart.removeItem(item.id)}
                  aria-label="Sil"
                >
                  <svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>

              <div class="flex items-center justify-between mt-2">
                <span class="text-xs font-semibold text-foreground">
                  {((item.priceCents * item.quantity) / 100).toLocaleString('tr-TR', { style: 'currency', currency: item.currency })}
                </span>

                <div class="flex items-center border border-border rounded-md bg-secondary/50">
                  <button
                    type="button"
                    class="w-6 h-6 flex items-center justify-center text-muted-foreground hover:text-foreground text-xs"
                    onclick={() => cart.updateQuantity(item.id, -1)}
                    aria-label="Azalt"
                  >
                    -
                  </button>
                  <span class="w-7 text-center text-xs font-mono font-medium">{item.quantity}</span>
                  <button
                    type="button"
                    class="w-6 h-6 flex items-center justify-center text-muted-foreground hover:text-foreground text-xs"
                    onclick={() => cart.updateQuantity(item.id, 1)}
                    aria-label="Artır"
                  >
                    +
                  </button>
                </div>
              </div>
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <!-- Footer -->
    {#if cart.items.length > 0}
      <div class="p-6 border-t border-border bg-card/80 backdrop-blur-xs space-y-4">
        <div class="space-y-1.5">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>Ara Toplam</span>
            <span>{cart.formattedTotal}</span>
          </div>
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>Kargo</span>
            <span class="text-emerald-500 font-medium">Ücretsiz</span>
          </div>
          <div class="flex items-center justify-between text-sm font-semibold text-foreground pt-2 border-t border-border">
            <span>Toplam</span>
            <span class="text-base font-bold text-foreground">{cart.formattedTotal}</span>
          </div>
        </div>

        <button
          type="button"
          class="w-full h-11 bg-primary text-primary-foreground font-medium rounded-lg hover:bg-primary/90 transition-colors flex items-center justify-center gap-2 text-sm shadow-xs"
          onclick={() => {
            close();
            oncheckout();
          }}
        >
          <span>Güvenli Ödemeye Geç</span>
          <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M14 5l7 7m0 0l-7 7m7-7H3" />
          </svg>
        </button>

        <p class="text-[11px] text-center text-muted-foreground flex items-center justify-center gap-1.5">
          <svg class="w-3.5 h-3.5 text-muted-foreground" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          Shopier 256-Bit SSL güvencesiyle 3D Secure ödeme
        </p>
      </div>
    {/if}
  </div>
{/if}

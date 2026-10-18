<script lang="ts">
  interface Props {
    phone: string;
    siteName?: string;
  }

  let { phone, siteName = 'HydroPost' }: Props = $props();

  let cleanPhone = $derived((phone || '').replace(/[^0-9]/g, ''));
  let message = $derived(`Merhaba ${siteName} Haber Merkezi, haber ihbarında bulunmak istiyorum:`);
  let whatsappUrl = $derived(
    `https://wa.me/${cleanPhone}?text=${encodeURIComponent(message)}`
  );
</script>

<!-- Floating WhatsApp Tip Button (Only renders when a valid phone number is configured) -->
{#if cleanPhone && cleanPhone.length >= 10}
  <a
    href={whatsappUrl}
    target="_blank"
    rel="noopener noreferrer"
    class="fixed bottom-5 right-5 z-40 flex items-center gap-2.5 px-4 py-2.5 rounded-full bg-emerald-700 hover:bg-emerald-800 text-white font-bold text-xs shadow-lg shadow-emerald-900/25 transition-all hover:-translate-y-0.5 active:translate-y-0 group"
    aria-label="WhatsApp Haber İhbar Hattı"
  >
    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M12.02 2C6.5 2 2 6.48 2 12c0 1.85.5 3.58 1.37 5.07L2 22l5.1-1.33A10 10 0 0 0 12.02 22C17.5 22 22 17.52 22 12S17.5 2 12.02 2Zm5.78 14.17c-.24.68-1.4 1.3-1.94 1.38-.5.08-1.12.11-1.8-.11-.42-.13-.95-.3-1.64-.58-2.88-1.24-4.76-4.13-4.9-4.33-.14-.2-1.17-1.56-1.17-2.97 0-1.42.74-2.11 1-2.4.26-.29.57-.36.76-.36h.54c.18 0 .42-.02.65.5.24.55.8 1.92.87 2.06.07.14.11.3.02.49-.09.19-.14.3-.28.46-.14.16-.29.36-.42.48-.14.14-.28.29-.12.57.16.28.7 1.17 1.52 1.9 1.05.94 1.93 1.23 2.21 1.37.28.14.44.12.6-.07.16-.19.68-.79.86-1.07.18-.28.36-.23.6-.14.24.09 1.53.73 1.8.86.26.14.44.2.5.3.07.12.07.65-.17 1.33Z"/>
    </svg>
    <span class="hidden sm:inline font-semibold">Haber İhbar</span>
  </a>
{/if}

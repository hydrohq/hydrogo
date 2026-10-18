<script lang="ts">
  interface Props {
    phone?: string;
    siteName?: string;
  }

  let { phone = '905550100000', siteName = 'HydroPost' }: Props = $props();

  let message = $derived(`Merhaba ${siteName} Haber Merkezi, haber ihbarında bulunmak istiyorum:`);
  let whatsappUrl = $derived(
    `https://wa.me/${phone.replace(/[^0-9]/g, '')}?text=${encodeURIComponent(message)}`
  );
</script>

<!-- Floating WhatsApp Tip Button -->
<a
  href={whatsappUrl}
  target="_blank"
  rel="noopener noreferrer"
  class="fixed bottom-5 right-5 z-40 flex items-center gap-2.5 px-3.5 py-2.5 rounded-full bg-emerald-600 hover:bg-emerald-500 text-white font-bold text-xs shadow-2xl shadow-emerald-600/40 border border-emerald-400/40 transition-all hover:scale-105 group"
  aria-label="WhatsApp Haber İhbar Hattı"
>
  <span class="text-lg">💬</span>
  <span class="hidden sm:inline font-semibold">Haber İhbar</span>
  <span class="w-2 h-2 rounded-full bg-emerald-300 animate-ping"></span>
</a>

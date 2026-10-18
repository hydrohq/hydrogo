<script lang="ts">
  import type { PageProps } from './$types';
  import { sanitizeHtml } from '#lib';

  let { data }: PageProps = $props();
  let m = $derived(data.masthead);
</script>

<svelte:head>
  <title>Künye ve İletişim — {m.site_name}</title>
  <meta name="description" content={`${m.site_name} Basın İlan Kurumu yasal künye, yönetim kadrosu ve iletişim bilgileri.`} />
</svelte:head>

<div class="max-w-4xl mx-auto py-8">
  <div class="border-b border-zinc-200 pb-4 mb-8">
    <span class="text-xs font-bold text-red-600 uppercase tracking-widest">Yasal Bildirim & BİK Uyumu</span>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-zinc-900 mt-1">Künye ve Yönetim</h1>
    <p class="text-sm text-zinc-500 mt-2">
      5187 sayılı Basın Kanunu ve Basın İlan Kurumu İnternet Haber Siteleri Yönetmeliği gereğince hazırlanmıştır.
    </p>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
    <!-- Corporate Identity -->
    <div class="bg-white border border-zinc-200 shadow-xs rounded-2xl p-6 space-y-4">
      <h2 class="text-base font-bold text-zinc-900 flex items-center gap-2 border-b border-zinc-200 pb-3">
        <span class="w-2.5 h-2.5 rounded-full bg-red-600"></span>
        Yayın ve Yönetim Kadrosu
      </h2>

      <div class="space-y-3 text-sm">
        <div>
          <span class="text-zinc-500 block text-xs font-medium">İmtiyaz Sahibi / Tüzel Kişi Temsilcisi</span>
          <span class="text-zinc-900 font-semibold">{m.publisher}</span>
        </div>

        <div>
          <span class="text-zinc-500 block text-xs font-medium">Sorumlu Yazı İşleri Müdürü</span>
          <span class="text-zinc-900 font-semibold">{m.managing_editor}</span>
        </div>

        {#if m.news_director}
          <div>
            <span class="text-zinc-500 block text-xs font-medium">Haber Müdürü</span>
            <span class="text-zinc-900 font-semibold">{m.news_director}</span>
          </div>
        {/if}

        {#if m.legal_counsel}
          <div>
            <span class="text-zinc-500 block text-xs font-medium">Hukuk Müşaviri</span>
            <span class="text-zinc-900 font-semibold">{m.legal_counsel}</span>
          </div>
        {/if}

        <div>
          <span class="text-zinc-500 block text-xs font-medium">Yer Sağlayıcı (Hosting)</span>
          <span class="text-zinc-900 font-semibold">{m.hosting_provider}</span>
        </div>

        {#if m.publication_type}
          <div>
            <span class="text-zinc-500 block text-xs font-medium">Yayın Türü</span>
            <span class="text-zinc-900 font-semibold">{m.publication_type}</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Contact & Legal Notification -->
    <div class="bg-white border border-zinc-200 shadow-xs rounded-2xl p-6 space-y-4">
      <h2 class="text-base font-bold text-zinc-900 flex items-center gap-2 border-b border-zinc-200 pb-3">
        <span class="w-2.5 h-2.5 rounded-full bg-red-600"></span>
        İletişim ve Tebligat Adresi
      </h2>

      <div class="space-y-3 text-sm">
        <div>
          <span class="text-zinc-500 block text-xs font-medium">Yönetim ve Tebligat Adresi</span>
          <p class="text-zinc-900 whitespace-pre-line">{m.address}</p>
        </div>

        <div>
          <span class="text-zinc-500 block text-xs font-medium">Telefon</span>
          <a href={`tel:${m.phone}`} class="text-zinc-900 hover:text-red-600 font-semibold transition-colors">
            {m.phone}
          </a>
        </div>

        <div>
          <span class="text-zinc-500 block text-xs font-medium">Kurumsal E-posta</span>
          <a href={`mailto:${m.email}`} class="text-zinc-900 hover:text-red-600 font-semibold transition-colors">
            {m.email}
          </a>
        </div>

        {#if m.kep_address}
          <div>
            <span class="text-zinc-500 block text-xs font-medium">KEP (Kayıtlı Elektronik Posta)</span>
            <code class="text-xs bg-zinc-100 px-2 py-1 rounded text-red-600 font-mono border border-zinc-200">
              {m.kep_address}
            </code>
          </div>
        {/if}
      </div>
    </div>
  </div>

  <!-- Editorial Principles -->
  {#if m.editorial_principles}
    <div class="bg-zinc-50 border border-zinc-200 rounded-2xl p-6">
      <h2 class="text-base font-bold text-zinc-900 mb-3">Yayın İlkeleri ve Gizlilik</h2>
      <div class="prose prose-sm text-zinc-700 leading-relaxed">
        {@html sanitizeHtml(m.editorial_principles)}
      </div>
    </div>
  {/if}
</div>

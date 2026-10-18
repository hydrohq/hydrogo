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

<div class="py-8">
  <div class="border-b border-[var(--hp-border)] pb-4 mb-8">
    <span class="text-xs font-bold text-[var(--hp-primary)] uppercase tracking-widest">Yasal Bildirim & BİK Uyumu</span>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-[var(--hp-ink)] mt-1">Künye ve Yönetim</h1>
    <p class="text-sm text-[var(--hp-muted)] mt-2">
      5187 sayılı Basın Kanunu ve Basın İlan Kurumu İnternet Haber Siteleri Yönetmeliği gereğince hazırlanmıştır.
    </p>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
    <!-- Corporate Identity -->
    <div class="bg-[var(--hp-surface)] border border-[var(--hp-border)] rounded-2xl p-6 space-y-4">
      <h2 class="text-base font-bold text-[var(--hp-ink)] flex items-center gap-2 border-b border-[var(--hp-border)] pb-3">
        <span class="w-2.5 h-2.5 rounded-full bg-[var(--hp-primary)]"></span>
        Yayın ve Yönetim Kadrosu
      </h2>

      <div class="space-y-3 text-sm">
        <div>
          <span class="text-[var(--hp-muted)] block text-xs font-medium">İmtiyaz Sahibi / Tüzel Kişi Temsilcisi</span>
          <span class="text-[var(--hp-ink)] font-semibold">{m.publisher}</span>
        </div>

        <div>
          <span class="text-[var(--hp-muted)] block text-xs font-medium">Sorumlu Yazı İşleri Müdürü</span>
          <span class="text-[var(--hp-ink)] font-semibold">{m.managing_editor}</span>
        </div>

        {#if m.news_director}
          <div>
            <span class="text-[var(--hp-muted)] block text-xs font-medium">Haber Müdürü</span>
            <span class="text-[var(--hp-ink)] font-semibold">{m.news_director}</span>
          </div>
        {/if}

        {#if m.legal_counsel}
          <div>
            <span class="text-[var(--hp-muted)] block text-xs font-medium">Hukuk Müşaviri</span>
            <span class="text-[var(--hp-ink)] font-semibold">{m.legal_counsel}</span>
          </div>
        {/if}

        <div>
          <span class="text-[var(--hp-muted)] block text-xs font-medium">Yer Sağlayıcı (Hosting)</span>
          <span class="text-[var(--hp-ink)] font-semibold">{m.hosting_provider}</span>
        </div>

        {#if m.publication_type}
          <div>
            <span class="text-[var(--hp-muted)] block text-xs font-medium">Yayın Türü</span>
            <span class="text-[var(--hp-ink)] font-semibold">{m.publication_type}</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Contact & Legal Notification -->
    <div class="bg-[var(--hp-surface)] border border-[var(--hp-border)] rounded-2xl p-6 space-y-4">
      <h2 class="text-base font-bold text-[var(--hp-ink)] flex items-center gap-2 border-b border-[var(--hp-border)] pb-3">
        <span class="w-2.5 h-2.5 rounded-full bg-[var(--hp-primary)]"></span>
        İletişim ve Tebligat Adresi
      </h2>

      <div class="space-y-3 text-sm">
        <div>
          <span class="text-[var(--hp-muted)] block text-xs font-medium">Yönetim ve Tebligat Adresi</span>
          <p class="text-[var(--hp-ink)] whitespace-pre-line">{m.address}</p>
        </div>

        <div>
          <span class="text-[var(--hp-muted)] block text-xs font-medium">Telefon</span>
          <a href={`tel:${m.phone}`} class="text-[var(--hp-ink)] hover:text-[var(--hp-primary)] font-semibold transition-colors">
            {m.phone}
          </a>
        </div>

        <div>
          <span class="text-[var(--hp-muted)] block text-xs font-medium">Kurumsal E-posta</span>
          <a href={`mailto:${m.email}`} class="text-[var(--hp-ink)] hover:text-[var(--hp-primary)] font-semibold transition-colors">
            {m.email}
          </a>
        </div>

        {#if m.kep_address}
          <div>
            <span class="text-[var(--hp-muted)] block text-xs font-medium">KEP (Kayıtlı Elektronik Posta)</span>
            <code class="text-xs bg-[var(--hp-surface-sunken)] px-2 py-1 rounded text-[var(--hp-primary)] font-mono border border-[var(--hp-border)]">
              {m.kep_address}
            </code>
          </div>
        {/if}
      </div>
    </div>
  </div>

  <!-- Editorial Principles -->
  {#if m.editorial_principles}
    <div class="bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] rounded-2xl p-6">
      <h2 class="text-base font-bold text-[var(--hp-ink)] mb-3">Yayın İlkeleri ve Gizlilik</h2>
      <div class="prose prose-sm text-[var(--hp-ink-soft)] leading-relaxed">
        {@html sanitizeHtml(m.editorial_principles)}
      </div>
    </div>
  {/if}
</div>

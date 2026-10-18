<script lang="ts">
  import type { PageProps } from './$types';
  import { sanitizeHtml } from '#lib';

  let { data }: PageProps = $props();
  let p = $derived(data.page);

  const formattedDate = $derived(
    p.updated_at
      ? new Date(p.updated_at).toLocaleDateString('tr-TR', {
          year: 'numeric',
          month: 'long',
          day: 'numeric',
        })
      : ''
  );
</script>

<svelte:head>
  <title>{p.title} — HydroPost</title>
  <meta name="description" content={`${p.title} - Yasal bilgilendirme ve kurumsal ilkeler.`} />
</svelte:head>

<div class="py-6 sm:py-8 max-w-4xl mx-auto">
  <!-- Breadcrumb -->
  <nav class="flex items-center gap-2 text-xs text-[var(--hp-muted)] mb-6" aria-label="Bağlantı yolu">
    <a href="/" class="hover:text-[var(--hp-primary)] transition-colors">Anasayfa</a>
    <span>/</span>
    <span>Kurumsal & Yasal</span>
    <span>/</span>
    <span class="text-[var(--hp-ink)] font-semibold truncate">{p.title}</span>
  </nav>

  <!-- Page Header -->
  <div class="border-b border-[var(--hp-border)] pb-6 mb-8">
    <span class="text-xs font-bold text-[var(--hp-primary)] uppercase tracking-wider">Kurumsal & Yasal Metin</span>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-[var(--hp-ink)] mt-2 tracking-tight">
      {p.title}
    </h1>
    {#if formattedDate}
      <p class="text-xs text-[var(--hp-muted)] mt-2">
        Son Güncelleme: <time class="font-medium text-[var(--hp-ink-soft)]">{formattedDate}</time>
      </p>
    {/if}
  </div>

  <!-- Page Body -->
  <div class="bg-[var(--hp-surface)] border border-[var(--hp-border)] rounded-2xl p-6 sm:p-10 shadow-sm">
    <article class="prose max-w-none text-[var(--hp-ink)] leading-relaxed space-y-4">
      {@html sanitizeHtml(p.body)}
    </article>
  </div>
</div>

<script lang="ts">
  import { onMount } from 'svelte';
  import type { PageProps } from './$types';
  import { sanitizeHtml, trackView, formatViews } from '#lib';

  let { data }: PageProps = $props();
  let p = $derived(data.page);
  let viewCount = $state<number>(0);
  $effect(() => {
    if (data.views) viewCount = data.views;
  });

  const formattedDate = $derived(
    p.updated_at
      ? new Date(p.updated_at).toLocaleDateString('tr-TR', {
          year: 'numeric',
          month: 'long',
          day: 'numeric',
        })
      : ''
  );

  onMount(async () => {
    if (p.slug) {
      const updated = await trackView('page_' + p.slug, 'page');
      if (updated !== null) {
        viewCount = updated;
      }
    }
  });
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
    <div class="flex items-center gap-4 text-xs text-[var(--hp-muted)] mt-2 flex-wrap">
      {#if formattedDate}
        <p>
          Son Güncelleme: <time class="font-medium text-[var(--hp-ink-soft)]">{formattedDate}</time>
        </p>
      {/if}
      <span class="flex items-center gap-1.5 font-medium text-[var(--hp-ink-soft)]">
        <svg class="w-3.5 h-3.5 text-[var(--hp-muted)]" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" />
          <circle cx="12" cy="12" r="3" />
        </svg>
        <span>{formatViews(viewCount)} okunma</span>
      </span>
    </div>
  </div>

  <!-- Page Body -->
  <div class="bg-[var(--hp-surface)] border border-[var(--hp-border)] rounded-2xl p-6 sm:p-10 shadow-sm">
    <article class="prose max-w-none text-[var(--hp-ink)] leading-relaxed space-y-4">
      {@html sanitizeHtml(p.body)}
    </article>
  </div>
</div>

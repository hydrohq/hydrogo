<script lang="ts">
  import type { PageProps } from './$types';
  import ColumnistsSection from '#lib/components/ColumnistsSection.svelte';

  let { data }: PageProps = $props();
</script>

<svelte:head>
  <title>Yazarlarımız — HydroPost</title>
  <meta name="description" content="HydroPost köşe yazarları ve güncel görüşleri." />
</svelte:head>

<div class="max-w-6xl mx-auto py-8">
  <div class="mb-8">
    <span class="text-xs font-bold text-[var(--hp-primary)] uppercase tracking-widest">Görüşler</span>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-[var(--hp-ink)] mt-1">Yazarlarımız</h1>
    <p class="text-sm text-[var(--hp-muted)] mt-1">Köşe yazarlarımız ve güncel yazıları.</p>
  </div>

  {#if data.columnists.length > 0}
    <ColumnistsSection columnists={data.columnists} />
  {:else}
    <div class="text-center py-16 rounded-xl bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)]">
      <p class="text-sm font-bold text-[var(--hp-ink)]">Henüz yazar eklenmedi</p>
    </div>
  {/if}
</div>

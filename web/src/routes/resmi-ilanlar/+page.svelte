<script lang="ts">
  import type { PageProps } from './$types';
  import { sanitizeHtml, safeUrl } from '#lib';

  let { data }: PageProps = $props();
</script>

<svelte:head>
  <title>Resmi İlanlar ve Duyurular — HydroPost</title>
  <meta name="description" content="Basın İlan Kurumu ve kamu kurumları resmi ihale, tebligat ve kamu personel ilanları." />
</svelte:head>

<div class="max-w-5xl mx-auto py-8">
  <div class="border-b border-zinc-800 pb-4 mb-8 flex flex-col sm:flex-row sm:items-end justify-between gap-4">
    <div>
      <span class="text-xs font-bold text-amber-500 uppercase tracking-widest">Basın İlan Kurumu</span>
      <h1 class="text-3xl sm:text-4xl font-extrabold text-white mt-1">Resmi İlanlar</h1>
      <p class="text-sm text-zinc-400 mt-1">
        Kamu kurum ve kuruluşlarına ait güncel ihale, tebligat ve yasal duyurular.
      </p>
    </div>

    <!-- Category filter tabs -->
    <div class="flex items-center gap-1.5 bg-zinc-900 p-1 rounded-xl border border-zinc-800 text-xs shrink-0 overflow-x-auto">
      {#each [
        { key: 'all', label: 'Tümü', href: '/resmi-ilanlar' },
        { key: 'ihale', label: 'İhale', href: '/resmi-ilanlar?tur=ihale' },
        { key: 'tebligat', label: 'Tebligat', href: '/resmi-ilanlar?tur=tebligat' },
        { key: 'icra', label: 'İcra / Satış', href: '/resmi-ilanlar?tur=icra' },
        { key: 'personel', label: 'Personel', href: '/resmi-ilanlar?tur=personel' },
      ] as tab}
        <a
          href={tab.href}
          class={`px-3 py-1.5 rounded-lg font-semibold transition-colors whitespace-nowrap ${
            data.currentFilter === tab.key
              ? 'bg-amber-500 text-black shadow'
              : 'text-zinc-400 hover:text-white hover:bg-zinc-800'
          }`}
        >
          {tab.label}
        </a>
      {/each}
    </div>
  </div>

  {#if data.announcements.length === 0}
    <div class="text-center py-16 bg-zinc-900/40 border border-zinc-800 rounded-2xl">
      <div class="text-4xl mb-3">📋</div>
      <h3 class="text-base font-bold text-white">Yayınlanmış resmi ilan bulunmamaktadır</h3>
      <p class="text-xs text-zinc-500 mt-1">Bu kategoride henüz aktif bir yasal duyuru veya ihale ilanı girilmemiştir.</p>
    </div>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each data.announcements as item}
        <div class="p-5 rounded-2xl bg-zinc-900/60 border border-zinc-800 hover:border-zinc-700 transition-all flex flex-col justify-between">
          <div class="space-y-3">
            <div class="flex items-center justify-between text-xs">
              <span class="px-2 py-0.5 rounded bg-amber-500/10 text-amber-400 font-bold uppercase text-[10px] border border-amber-500/20">
                {item.announcement_type || 'Resmi İlan'}
              </span>
              {#if item.bik_number}
                <span class="text-zinc-500 font-mono text-[11px]">BİK: {item.bik_number}</span>
              {/if}
            </div>

            <h2 class="text-base font-bold text-white leading-snug">
              {item.title}
            </h2>

            <div class="text-xs text-zinc-400 flex items-center gap-2">
              <span class="font-semibold text-zinc-300">{item.institution}</span>
              {#if item.city}
                <span>• {item.city}</span>
              {/if}
            </div>

            <div class="text-xs text-zinc-400 line-clamp-3 leading-relaxed border-t border-zinc-800/80 pt-2">
              {@html sanitizeHtml(item.body)}
            </div>
          </div>

          <div class="mt-4 pt-3 border-t border-zinc-800/80 flex items-center justify-between text-xs">
            {#if item.deadline}
              <span class="text-amber-400 font-semibold text-[11px]">
                Son Tarih: {item.deadline}
              </span>
            {:else}
              <span class="text-zinc-500 text-[11px]">Aktif İlan</span>
            {/if}

            {#if item.official_url}
              <a
                href={safeUrl(item.official_url)}
                target="_blank"
                rel="noopener noreferrer"
                class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold text-[11px] transition-colors"
              >
                İlan.gov.tr ↗
              </a>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<script lang="ts">
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let currentCity = $derived(data.city);

  const cities = ['İstanbul', 'Ankara', 'İzmir', 'Bursa', 'Antalya', 'Adana', 'Konya', 'Trabzon'];
</script>

<svelte:head>
  <title>{currentCity} Nöbetçi Eczaneler — HydroPost</title>
  <meta name="description" content={`Bugün ${currentCity} genelinde açık olan nöbetçi eczaneler, adres, telefon ve yol tarifi bilgileri.`} />
</svelte:head>

<div class="max-w-6xl mx-auto py-8">
  <div class="border-b border-zinc-800 pb-4 mb-8 flex flex-col sm:flex-row sm:items-end justify-between gap-4">
    <div>
      <span class="text-xs font-bold text-red-500 uppercase tracking-widest">Sağlık & Yerel Rehber</span>
      <h1 class="text-3xl sm:text-4xl font-extrabold text-white mt-1">{currentCity} Nöbetçi Eczaneleri</h1>
      <p class="text-sm text-zinc-400 mt-1">
        Bugün akşamdan yarın sabaha kadar kesintisiz hizmet veren nöbetçi eczanelerin güncel listesi.
      </p>
    </div>

    <!-- City Selector buttons -->
    <div class="flex items-center gap-1.5 overflow-x-auto p-1 bg-zinc-900 border border-zinc-800 rounded-xl text-xs">
      {#each cities as c}
        <a
          href={`/nobetci-eczaneler?sehir=${encodeURIComponent(c)}`}
          class={`px-3 py-1.5 rounded-lg font-semibold whitespace-nowrap transition-colors ${
            data.city === c ? 'bg-red-600 text-white' : 'text-zinc-400 hover:text-white hover:bg-zinc-800'
          }`}
        >
          {c}
        </a>
      {/each}
    </div>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
    <!-- Pharmacies List (2 columns on md) -->
    <div class="md:col-span-2 space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-bold text-white flex items-center gap-2">
          <span class="w-2.5 h-2.5 rounded-full bg-red-600 animate-pulse"></span>
          Aktif Nöbetçi Eczaneler ({data.pharmacies.length})
        </h2>
        <span class="text-xs text-zinc-500">Bugün 18:00 - Yarın 08:30</span>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {#each data.pharmacies as p}
          <div class="p-5 rounded-2xl bg-zinc-900/60 border border-zinc-800 hover:border-zinc-700 transition-all flex flex-col justify-between">
            <div class="space-y-2">
              <div class="flex items-center justify-between text-xs">
                <span class="px-2 py-0.5 rounded bg-red-950/60 text-red-400 font-bold border border-red-900/40 text-[11px]">
                  {p.district}
                </span>
                <span class="text-emerald-400 font-semibold text-[11px] flex items-center gap-1">
                  ● Açık
                </span>
              </div>

              <h3 class="text-base font-bold text-white">{p.name}</h3>
              <p class="text-xs text-zinc-300 leading-relaxed">{p.address}</p>

              {#if p.note}
                <div class="text-[11px] text-zinc-500 bg-zinc-950/80 p-2 rounded-lg border border-zinc-800/80">
                  📍 Tarif: {p.note}
                </div>
              {/if}
            </div>

            <div class="mt-4 pt-3 border-t border-zinc-800 flex items-center justify-between text-xs">
              <a
                href={`tel:${p.phone.replace(/[^0-9]/g, '')}`}
                class="font-mono text-zinc-200 hover:text-red-400 font-bold transition-colors"
              >
                📞 {p.phone}
              </a>
              <a
                href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(`${p.name} ${p.address} ${p.city}`)}`}
                target="_blank"
                rel="noopener noreferrer"
                class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold text-[11px] transition-colors"
              >
                Yol Tarifi ↗
              </a>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <!-- Sidebar: Prayer Times & Weather for this city -->
    <div class="space-y-6">
      <!-- Prayer Times Card -->
      <div class="p-6 rounded-2xl bg-zinc-900/60 border border-zinc-800 space-y-4">
        <h3 class="text-sm font-bold text-white flex items-center justify-between border-b border-zinc-800 pb-3">
          <span class="flex items-center gap-2">
            <span>🕌</span> {data.city} Namaz Vakitleri
          </span>
          <span class="text-xs font-normal text-amber-400 font-mono">
            {data.prayer.nextPrayer} {data.prayer.remainingTime}
          </span>
        </h3>

        <div class="grid grid-cols-2 gap-2 text-xs">
          <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800/60 flex items-center justify-between">
            <span class="text-zinc-400">İmsak</span>
            <span class="font-bold text-zinc-100 font-mono">{data.prayer.fajr}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800/60 flex items-center justify-between">
            <span class="text-zinc-400">Güneş</span>
            <span class="font-bold text-zinc-100 font-mono">{data.prayer.sunrise}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800/60 flex items-center justify-between">
            <span class="text-zinc-400">Öğle</span>
            <span class="font-bold text-zinc-100 font-mono">{data.prayer.dhuhr}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800/60 flex items-center justify-between">
            <span class="text-zinc-400">İkindi</span>
            <span class="font-bold text-zinc-100 font-mono">{data.prayer.asr}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800/60 flex items-center justify-between">
            <span class="text-zinc-400">Akşam</span>
            <span class="font-bold text-zinc-100 font-mono">{data.prayer.maghrib}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-zinc-950 border border-zinc-800/60 flex items-center justify-between">
            <span class="text-zinc-400">Yatsı</span>
            <span class="font-bold text-zinc-100 font-mono">{data.prayer.isha}</span>
          </div>
        </div>
      </div>

      <!-- Weather Card -->
      <div class="p-6 rounded-2xl bg-zinc-900/60 border border-zinc-800 space-y-3">
        <h3 class="text-sm font-bold text-white flex items-center gap-2 border-b border-zinc-800 pb-3">
          <span>⛅</span> {data.city} Hava Durumu
        </h3>

        <div class="flex items-center justify-between py-2">
          <div class="flex items-center gap-3">
            <span class="text-4xl">{data.weather.icon}</span>
            <div>
              <div class="text-2xl font-black text-white">{data.weather.temp}°C</div>
              <div class="text-xs text-zinc-400">{data.weather.condition}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

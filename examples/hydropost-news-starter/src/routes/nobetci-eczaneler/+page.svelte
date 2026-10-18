<script lang="ts">
  import { goto } from '$app/navigation';
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let currentCity = $derived(data.city);

  // All 81 provinces — only a handful have curated demo pharmacy/weather/
  // prayer-time data (see lib/services/infoWidgets.ts); the rest fall back
  // to İstanbul's demo data, same as any unrecognized city already did.
  const cities = [
    'Adana', 'Adıyaman', 'Afyonkarahisar', 'Ağrı', 'Amasya', 'Ankara', 'Antalya', 'Artvin',
    'Aydın', 'Balıkesir', 'Bilecik', 'Bingöl', 'Bitlis', 'Bolu', 'Burdur', 'Bursa',
    'Çanakkale', 'Çankırı', 'Çorum', 'Denizli', 'Diyarbakır', 'Edirne', 'Elazığ', 'Erzincan',
    'Erzurum', 'Eskişehir', 'Gaziantep', 'Giresun', 'Gümüşhane', 'Hakkâri', 'Hatay', 'Isparta',
    'Mersin', 'İstanbul', 'İzmir', 'Kars', 'Kastamonu', 'Kayseri', 'Kırklareli', 'Kırşehir',
    'Kocaeli', 'Konya', 'Kütahya', 'Malatya', 'Manisa', 'Kahramanmaraş', 'Mardin', 'Muğla',
    'Muş', 'Nevşehir', 'Niğde', 'Ordu', 'Rize', 'Sakarya', 'Samsun', 'Siirt',
    'Sinop', 'Sivas', 'Tekirdağ', 'Tokat', 'Trabzon', 'Tunceli', 'Şanlıurfa', 'Uşak',
    'Van', 'Yozgat', 'Zonguldak', 'Aksaray', 'Bayburt', 'Karaman', 'Kırıkkale', 'Batman',
    'Şırnak', 'Bartın', 'Ardahan', 'Iğdır', 'Yalova', 'Karabük', 'Kilis', 'Osmaniye', 'Düzce',
  ].sort((a, b) => a.localeCompare(b, 'tr'));

  function onCityChange(e: Event) {
    const city = (e.target as HTMLSelectElement).value;
    goto(`/nobetci-eczaneler?sehir=${encodeURIComponent(city)}`);
  }
</script>

<svelte:head>
  <title>{currentCity} Nöbetçi Eczaneler — HydroPost</title>
  <meta name="description" content={`Bugün ${currentCity} genelinde açık olan nöbetçi eczaneler, adres, telefon ve yol tarifi bilgileri.`} />
</svelte:head>

<div class="py-8">
  <div class="border-b border-[var(--hp-border)] pb-4 mb-8 flex flex-col sm:flex-row sm:items-end justify-between gap-4">
    <div>
      <span class="text-xs font-bold text-[var(--hp-primary)] uppercase tracking-widest">Sağlık & Yerel Rehber</span>
      <h1 class="text-3xl sm:text-4xl font-extrabold text-[var(--hp-ink)] mt-1">{currentCity} Nöbetçi Eczaneleri</h1>
      <p class="text-sm text-[var(--hp-muted)] mt-1">
        Bugün akşamdan yarın sabaha kadar kesintisiz hizmet veren nöbetçi eczanelerin güncel listesi.
      </p>
    </div>

    <!-- City Selector -->
    <label class="relative inline-flex items-center shrink-0">
      <svg class="absolute left-3 w-4 h-4 text-[var(--hp-muted)] pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 21s7-6.5 7-11.5a7 7 0 1 0-14 0C5 14.5 12 21 12 21Z" stroke-linecap="round" stroke-linejoin="round" /><circle cx="12" cy="9.5" r="2.5" />
      </svg>
      <select
        value={currentCity}
        onchange={onCityChange}
        class="appearance-none pl-9 pr-9 py-2.5 rounded-lg bg-[var(--hp-surface)] border border-[var(--hp-border-strong)] text-sm font-semibold text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors cursor-pointer"
      >
        {#each cities as c}
          <option value={c}>{c}</option>
        {/each}
      </select>
      <svg class="absolute right-3 w-3.5 h-3.5 text-[var(--hp-muted)] pointer-events-none" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="m6 9 6 6 6-6" stroke-linecap="round" stroke-linejoin="round" />
      </svg>
    </label>
  </div>

  <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
    <!-- Pharmacies List (2 columns on md) -->
    <div class="md:col-span-2 space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-base font-bold text-[var(--hp-ink)] flex items-center gap-2">
          <span class="w-2.5 h-2.5 rounded-full bg-[var(--hp-primary)]"></span>
          Aktif Nöbetçi Eczaneler ({data.pharmacies.length})
        </h2>
        <span class="text-xs text-[var(--hp-muted)]">Bugün 18:00 - Yarın 08:30</span>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {#each data.pharmacies as p}
          <div class="p-5 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] flex flex-col justify-between">
            <div class="space-y-2">
              <div class="flex items-center justify-between text-xs">
                <span class="px-2 py-0.5 rounded bg-[var(--hp-primary-light)] text-[var(--hp-primary)] font-bold border border-[var(--hp-primary-border)] text-[11px]">
                  {p.district}
                </span>
                <span class="text-emerald-600 font-semibold text-[11px] flex items-center gap-1">
                  ● Açık
                </span>
              </div>

              <h3 class="text-base font-bold text-[var(--hp-ink)]">{p.name}</h3>
              <p class="text-xs text-[var(--hp-muted)] leading-relaxed">{p.address}</p>

              {#if p.note}
                <div class="text-[11px] text-[var(--hp-muted)] bg-[var(--hp-surface-sunken)] p-2 rounded-lg border border-[var(--hp-border)]">
                  📍 Tarif: {p.note}
                </div>
              {/if}
            </div>

            <div class="mt-4 pt-3 border-t border-[var(--hp-border)] flex items-center justify-between text-xs">
              <a
                href={`tel:${p.phone.replace(/[^0-9]/g, '')}`}
                class="font-mono text-[var(--hp-ink)] hover:text-[var(--hp-primary)] font-bold transition-colors"
              >
                📞 {p.phone}
              </a>
              <a
                href={`https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(`${p.name} ${p.address} ${p.city}`)}`}
                target="_blank"
                rel="noopener noreferrer"
                class="px-2.5 py-1 rounded bg-[var(--hp-surface-sunken)] hover:bg-[var(--hp-border)] text-[var(--hp-ink-soft)] font-semibold text-[11px] transition-colors"
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
      <div class="p-6 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] space-y-4">
        <h3 class="text-sm font-bold text-[var(--hp-ink)] flex items-center justify-between border-b border-[var(--hp-border)] pb-3">
          <span class="flex items-center gap-2">
            <span>🕌</span> {data.city} Namaz Vakitleri
          </span>
          <span class="text-xs font-normal text-amber-600 font-mono">
            {data.prayer.nextPrayer} {data.prayer.remainingTime}
          </span>
        </h3>

        <div class="grid grid-cols-2 gap-2 text-xs">
          <div class="p-2.5 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] flex items-center justify-between">
            <span class="text-[var(--hp-muted)]">İmsak</span>
            <span class="font-bold text-[var(--hp-ink)] font-mono">{data.prayer.fajr}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] flex items-center justify-between">
            <span class="text-[var(--hp-muted)]">Güneş</span>
            <span class="font-bold text-[var(--hp-ink)] font-mono">{data.prayer.sunrise}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] flex items-center justify-between">
            <span class="text-[var(--hp-muted)]">Öğle</span>
            <span class="font-bold text-[var(--hp-ink)] font-mono">{data.prayer.dhuhr}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] flex items-center justify-between">
            <span class="text-[var(--hp-muted)]">İkindi</span>
            <span class="font-bold text-[var(--hp-ink)] font-mono">{data.prayer.asr}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] flex items-center justify-between">
            <span class="text-[var(--hp-muted)]">Akşam</span>
            <span class="font-bold text-[var(--hp-ink)] font-mono">{data.prayer.maghrib}</span>
          </div>
          <div class="p-2.5 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] flex items-center justify-between">
            <span class="text-[var(--hp-muted)]">Yatsı</span>
            <span class="font-bold text-[var(--hp-ink)] font-mono">{data.prayer.isha}</span>
          </div>
        </div>
      </div>

      <!-- Weather Card -->
      <div class="p-6 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] space-y-3">
        <h3 class="text-sm font-bold text-[var(--hp-ink)] flex items-center gap-2 border-b border-[var(--hp-border)] pb-3">
          <span>⛅</span> {data.city} Hava Durumu
        </h3>

        <div class="flex items-center justify-between py-2">
          <div class="flex items-center gap-3">
            <span class="text-4xl">{data.weather.icon}</span>
            <div>
              <div class="text-2xl font-black text-[var(--hp-ink)]">{data.weather.temp}°C</div>
              <div class="text-xs text-[var(--hp-muted)]">{data.weather.condition}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

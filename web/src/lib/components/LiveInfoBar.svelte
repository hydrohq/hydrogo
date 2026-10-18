<script lang="ts">
  import { getLiveMarkets, getPrayerTimes, getWeather } from '#lib';

  let selectedCity = $state('İstanbul');
  let markets = $derived(getLiveMarkets());
  let prayer = $derived(getPrayerTimes(selectedCity));
  let weather = $derived(getWeather(selectedCity));
</script>

<div class="bg-zinc-50 border-b border-zinc-200 text-xs py-2 px-4 select-none">
  <div class="max-w-7xl mx-auto flex flex-col lg:flex-row items-center justify-between gap-3">
    <!-- Markets Ticker -->
    <div class="flex items-center gap-4 overflow-x-auto w-full lg:w-auto scrollbar-none py-0.5">
      <span class="text-zinc-500 font-bold uppercase text-[10px] tracking-wider shrink-0 flex items-center gap-1">
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-600 animate-pulse"></span>
        Piyasalar:
      </span>

      {#each markets as m}
        <div class="flex items-center gap-1.5 shrink-0 font-mono text-[11px] bg-white px-2 py-0.5 rounded border border-zinc-200 shadow-2xs">
          <span class="text-zinc-500 font-sans font-semibold">{m.name}</span>
          <span class="text-zinc-900 font-bold">{m.selling}</span>
          <span class={`text-[10px] font-bold ${m.changeRate >= 0 ? 'text-emerald-600' : 'text-red-600'}`}>
            {m.changeRate >= 0 ? `▲ %${m.changeRate}` : `▼ %${Math.abs(m.changeRate)}`}
          </span>
        </div>
      {/each}
    </div>

    <!-- Weather, Prayer & Pharmacy Row -->
    <div class="flex items-center gap-4 text-xs shrink-0 w-full lg:w-auto justify-between lg:justify-end border-t lg:border-t-0 border-zinc-200 pt-2 lg:pt-0">
      <!-- City selector -->
      <select
        bind:value={selectedCity}
        class="bg-white border border-zinc-300 text-zinc-700 rounded px-2 py-0.5 text-[11px] font-medium outline-none focus:border-red-500"
      >
        <option value="İstanbul">İstanbul</option>
        <option value="Ankara">Ankara</option>
        <option value="İzmir">İzmir</option>
        <option value="Bursa">Bursa</option>
        <option value="Antalya">Antalya</option>
      </select>

      <!-- Weather -->
      <div class="flex items-center gap-1.5 text-zinc-700">
        <span>{weather.icon}</span>
        <span class="font-bold text-zinc-900">{weather.temp}°C</span>
        <span class="text-zinc-500 hidden sm:inline text-[11px]">{weather.condition}</span>
      </div>

      <!-- Prayer Times Countdown -->
      <div class="flex items-center gap-1.5 text-zinc-700 border-l border-zinc-200 pl-3">
        <span>🕌</span>
        <span class="text-[11px] text-zinc-600">
          {prayer.nextPrayer}: <b class="text-zinc-900">{prayer.asr}</b>
          <span class="text-zinc-500 text-[10px] hidden sm:inline">({prayer.remainingTime} kaldı)</span>
        </span>
      </div>

      <!-- Duty Pharmacy Quick Link -->
      <a
        href="/nobetci-eczaneler"
        class="flex items-center gap-1 text-[11px] font-bold text-red-600 hover:text-red-700 bg-red-50 border border-red-200 px-2 py-0.5 rounded transition-colors whitespace-nowrap"
      >
        <span>🏥</span> Nöbetçi Eczaneler
      </a>
    </div>
  </div>
</div>

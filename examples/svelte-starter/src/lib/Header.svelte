<script lang="ts">
  interface Props {
    currentRoute?: string;
    onnavigate?: (route: string) => void;
  }

  let { currentRoute = '/', onnavigate }: Props = $props();
</script>

<header class="border-b border-slate-900 bg-slate-950/80 backdrop-blur sticky top-0 z-50">
  <div class="max-w-6xl mx-auto px-6 h-14 flex items-center justify-between">
    <button
      type="button"
      onclick={() => onnavigate?.('/')}
      class="flex items-center gap-2.5 text-slate-100 font-bold text-base hover:opacity-90 transition-opacity cursor-pointer bg-transparent border-0 p-0"
    >
      <div class="w-7 h-7 rounded-lg bg-sky-500/10 border border-sky-500/25 flex items-center justify-center text-sky-400 font-bold text-xs">
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
          <polyline points="2 17 12 22 22 17"></polyline>
          <polyline points="2 12 12 17 22 12"></polyline>
        </svg>
      </div>
      <span>Hydro<span class="text-sky-400">Go</span></span>
    </button>

    <nav class="flex items-center gap-4 text-xs font-medium">
      <button
        type="button"
        onclick={() => onnavigate?.('/docs')}
        class="transition-colors cursor-pointer bg-transparent border-0 {currentRoute === '/docs' ? 'text-white font-semibold' : 'text-slate-400 hover:text-white'}"
      >
        Docs
      </button>

      <a
        href="/admin/"
        class="text-slate-400 hover:text-white transition-colors"
      >
        Admin Studio
      </a>

      <a
        href="https://ardalabs.dev"
        target="_blank"
        rel="noreferrer"
        class="px-3 py-1.5 rounded-full bg-slate-900 border border-slate-800 text-slate-300 hover:text-white hover:border-slate-700 transition-colors flex items-center gap-1.5"
      >
        Visit Website
        <span aria-hidden="true">↗</span>
      </a>
    </nav>
  </div>
</header>

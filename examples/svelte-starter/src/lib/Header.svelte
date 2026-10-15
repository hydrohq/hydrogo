<script lang="ts">
  interface Props {
    currentRoute: string;
    onnavigate: (route: string) => void;
  }

  let { currentRoute, onnavigate }: Props = $props();
</script>

<header class="border-b border-slate-800 bg-slate-950/80 backdrop-blur sticky top-0 z-50">
  <div class="max-w-6xl mx-auto px-6 h-16 flex items-center justify-between">
    <!-- Brand -->
    <button
      type="button"
      onclick={() => onnavigate('/')}
      class="flex items-center gap-2.5 text-slate-100 font-bold text-lg hover:opacity-90 transition-opacity"
    >
      <div class="w-8 h-8 rounded-lg bg-sky-500/10 border border-sky-500/25 flex items-center justify-center text-sky-400 font-bold text-sm">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
          <polyline points="2 17 12 22 22 17"></polyline>
          <polyline points="2 12 12 17 22 12"></polyline>
        </svg>
      </div>
      <span>Hydro<span class="text-sky-400">Go</span></span>
      <span class="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-400 font-mono font-normal">Headless Starter</span>
    </button>

    <!-- Navigation -->
    <nav class="flex items-center gap-1">
      <button
        type="button"
        onclick={() => onnavigate('/')}
        class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {currentRoute === '/' ? 'text-sky-400 bg-sky-500/10' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'}"
      >
        Home
      </button>
      <button
        type="button"
        onclick={() => onnavigate('/news')}
        class="px-3.5 py-1.5 rounded-lg text-sm font-medium transition-colors {currentRoute.startsWith('/news') ? 'text-sky-400 bg-sky-500/10' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-900'}"
      >
        News
      </button>
      <a
        href="/admin"
        target="_blank"
        rel="noreferrer"
        class="ml-3 px-3 py-1.5 rounded-lg text-xs font-semibold bg-slate-800 hover:bg-slate-700 text-slate-200 transition-colors"
      >
        Admin Panel ↗
      </a>
    </nav>
  </div>
</header>

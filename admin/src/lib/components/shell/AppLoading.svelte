<script lang="ts">
  interface Props {
    message?: string;
  }

  let { message = 'Loading HydroGo...' }: Props = $props();
</script>

<div
  class="relative flex h-screen w-full flex-col items-center justify-center overflow-hidden bg-background text-foreground select-none font-sans"
  role="status"
  aria-live="polite"
  aria-label="Loading HydroGo Admin"
>
  <!-- Ambient background glow mesh -->
  <div
    class="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden"
    aria-hidden="true"
  >
    <div
      class="h-[380px] w-[380px] rounded-full bg-gradient-to-tr from-sky-500/15 via-indigo-500/10 to-transparent blur-3xl opacity-75 animate-pulse"
      style="animation-duration: 4s;"
    ></div>
    <div
      class="absolute h-[220px] w-[220px] rounded-full bg-sky-400/10 blur-2xl -translate-y-6"
    ></div>
  </div>

  <!-- Central Card Shell -->
  <div class="relative z-10 flex flex-col items-center max-w-xs w-full px-6 py-8 rounded-2xl border border-border/50 bg-card/60 backdrop-blur-xl shadow-2xl shadow-sky-500/5 space-y-6 animate-in fade-in zoom-in-95 duration-300">
    <!-- Glowing Brand Icon -->
    <div class="relative group">
      <div class="absolute -inset-2 rounded-2xl bg-gradient-to-r from-sky-500/30 via-indigo-500/30 to-sky-400/30 blur-md opacity-70 animate-pulse"></div>
      <div class="relative flex size-14 items-center justify-center rounded-xl bg-gradient-to-b from-primary via-primary/95 to-primary/85 text-primary-foreground font-bold shadow-lg shadow-primary/20 border border-white/20">
        <svg
          class="size-7 drop-shadow-sm"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
          <polyline points="2 17 12 22 22 17"></polyline>
          <polyline points="2 12 12 17 22 12"></polyline>
        </svg>
      </div>
    </div>

    <!-- Title & Brand Text -->
    <div class="space-y-1 text-center">
      <div class="flex items-center justify-center gap-1.5">
        <span class="text-xl font-bold tracking-tight text-foreground">
          Hydro<span class="text-sky-400 font-extrabold">Go</span>
        </span>
        <span class="inline-flex items-center rounded-full bg-primary/10 border border-primary/20 px-2 py-0.5 text-[10px] font-mono font-medium text-sky-400">
          Studio
        </span>
      </div>
      <p class="text-xs text-muted-foreground font-medium">Headless Content Management</p>
    </div>

    <!-- Modern Indeterminate Shimmer Progress Bar -->
    <div class="w-44 space-y-3">
      <div class="relative h-1 w-full overflow-hidden rounded-full bg-muted/60 border border-border/40">
        <div class="hydro-indeterminate-bar absolute inset-y-0 h-full rounded-full bg-gradient-to-r from-sky-400 via-indigo-500 to-sky-300"></div>
      </div>

      <!-- Live Status Pill -->
      <div class="flex items-center justify-center gap-2 text-muted-foreground">
        <span class="relative flex size-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-sky-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full size-2 bg-sky-500"></span>
        </span>
        <span class="font-mono text-[11px] uppercase tracking-wider text-muted-foreground/90 font-medium">
          {message}
        </span>
      </div>
    </div>
  </div>

  <!-- Bottom subtle version info -->
  <div class="absolute bottom-6 flex items-center gap-2 text-[11px] font-mono text-muted-foreground/50">
    <span>HydroGo v2.0</span>
    <span>•</span>
    <span>Single-Binary CMS</span>
  </div>
</div>

<style>
  .hydro-indeterminate-bar {
    width: 45%;
    animation: hydro-slide 1.4s cubic-bezier(0.4, 0, 0.2, 1) infinite;
  }

  @keyframes hydro-slide {
    0% {
      left: -45%;
    }
    100% {
      left: 100%;
    }
  }
</style>

<script lang="ts">
  import BrandLogo from './BrandLogo.svelte';

  interface Props {
    onopenpage?: (slug: string) => void;
    onopennews?: (slug: string) => void;
    onopendocs?: () => void;
  }

  let { onopenpage, onopennews, onopendocs }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center min-h-[78vh] text-center px-4 max-w-xl mx-auto space-y-9 sm:space-y-10">
  <!-- Brand Logo -->
  <div class="select-none flex items-center justify-center">
    <BrandLogo size="xl" />
  </div>

  <!-- Getting Started Steps (Clean list without card background, exactly like Next.js starter) -->
  <ol class="list-decimal list-inside space-y-2.5 text-sm sm:text-base text-zinc-300 font-sans text-left leading-relaxed">
    <li>
      Get started by editing <code class="bg-white/10 text-white font-mono text-xs sm:text-sm px-2 py-0.5 rounded font-semibold border border-white/5">src/App.svelte</code>
    </li>
    <li>
      Save and see your changes instantly.
    </li>
    <li>
      Manage content and media in <a href="/admin/" class="text-sky-400 hover:text-sky-300 underline underline-offset-4"><code class="bg-white/10 text-sky-300 font-mono text-xs sm:text-sm px-2 py-0.5 rounded font-semibold border border-white/5">/admin</code></a>.
    </li>
  </ol>

  <!-- Action Buttons -->
  <div class="flex flex-col sm:flex-row items-center justify-center gap-4 w-full">
    <!-- Visit Our Website (White pill button with triangle icon) -->
    <a
      href="https://ardalabs.dev"
      target="_blank"
      rel="noreferrer"
      class="w-full sm:w-auto h-11 px-7 rounded-full bg-white hover:bg-zinc-200 text-black font-semibold text-sm flex items-center justify-center gap-2 transition-colors shadow-sm cursor-pointer"
    >
      <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
        <polygon points="12 2 22 22 2 22"></polygon>
      </svg>
      Visit Our Website
    </a>

    <!-- Read our docs (Dark pill button) -->
    <button
      type="button"
      onclick={onopendocs}
      class="w-full sm:w-auto h-11 px-7 rounded-full border border-white/20 hover:border-white/40 bg-black hover:bg-zinc-900 text-white font-medium text-sm flex items-center justify-center gap-2 transition-colors cursor-pointer"
    >
      Read our docs
    </button>
  </div>

  <!-- Bottom Links Row (Like Next.js footer) -->
  <div class="pt-6 sm:pt-8 flex flex-wrap items-center justify-center gap-6 sm:gap-8 text-xs sm:text-sm text-zinc-400 font-medium">
    <button
      type="button"
      onclick={onopendocs}
      class="hover:text-white transition-colors flex items-center gap-2 cursor-pointer bg-transparent border-0 p-0 text-inherit"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
        <polyline points="14 2 14 8 20 8"></polyline>
        <line x1="16" y1="13" x2="8" y2="13"></line>
        <line x1="16" y1="17" x2="8" y2="17"></line>
      </svg>
      Read our docs
    </button>

    <a
      href="/admin/"
      class="hover:text-white transition-colors flex items-center gap-2"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
        <line x1="3" y1="9" x2="21" y2="9"></line>
        <line x1="9" y1="21" x2="9" y2="9"></line>
      </svg>
      Admin Studio
    </a>

    <a
      href="https://ardalabs.dev"
      target="_blank"
      rel="noreferrer"
      class="hover:text-white transition-colors flex items-center gap-1.5"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="12" cy="12" r="10"></circle>
        <line x1="2" y1="12" x2="22" y2="12"></line>
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
      </svg>
      Visit Our Website
      <span aria-hidden="true" class="text-xs">→</span>
    </a>
  </div>
</div>

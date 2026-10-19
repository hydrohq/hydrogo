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

    <!-- Read our docs (Dark pill button) -->
    <button
      type="button"
      onclick={onopendocs}
      class="w-full sm:w-auto h-11 px-7 rounded-full border border-white/20 hover:border-white/40 bg-black hover:bg-zinc-900 text-white font-medium text-sm flex items-center justify-center gap-2 transition-colors cursor-pointer"
    >
      Read our docs
    </button>
  </div>
</div>

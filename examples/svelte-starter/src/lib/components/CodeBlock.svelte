<script lang="ts">
  import { highlight, languageLabel } from '../highlight';

  let { code, lang = 'text', title = '' }: { code: string; lang?: string; title?: string } = $props();

  let copied = $state(false);
  let timer: ReturnType<typeof setTimeout> | undefined;
  const html = $derived(highlight(code, lang));

  async function copy() {
    try {
      await navigator.clipboard.writeText(code);
    } catch {
      const area = document.createElement('textarea');
      area.value = code;
      area.setAttribute('readonly', '');
      area.style.position = 'fixed';
      area.style.opacity = '0';
      document.body.appendChild(area);
      area.select();
      document.execCommand('copy');
      area.remove();
    }
    copied = true;
    clearTimeout(timer);
    timer = setTimeout(() => (copied = false), 1800);
  }
</script>

<div class="group my-5 overflow-hidden rounded-lg border border-line" style="background: var(--code-bg)">
  <div class="flex items-center justify-between gap-3 border-b border-line px-3 py-1.5 text-xs" style="background: var(--code-head)">
    <span class="flex min-w-0 items-center gap-2 text-muted">
      <span class="rounded bg-accent-soft px-1.5 py-0.5 font-medium text-accent">{languageLabel(lang)}</span>
      {#if title}<span class="truncate font-mono">{title}</span>{/if}
    </span>
    <button
      type="button"
      onclick={copy}
      class="inline-flex shrink-0 items-center gap-1.5 rounded px-2 py-1 font-medium text-muted hover:bg-bg hover:text-fg"
      aria-label={copied ? 'Copied to clipboard' : `Copy ${languageLabel(lang)} code`}
    >
      {#if copied}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="20 6 9 17 4 12" /></svg>
        Copied
      {:else}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="9" y="9" width="13" height="13" rx="2" /><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" /></svg>
        Copy
      {/if}
    </button>
  </div>
  <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
  <pre class="thin-scroll m-0 overflow-x-auto p-4 font-mono text-[13px] leading-relaxed" style="color: var(--code-fg)" tabindex="0"><code>{@html html}</code></pre>
  <span class="sr-only" aria-live="polite">{copied ? 'Copied to clipboard' : ''}</span>
</div>

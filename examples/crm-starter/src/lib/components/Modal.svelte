<script lang="ts">
  import type { Snippet } from 'svelte';
  import { fade, fly } from 'svelte/transition';
  import { ease, ms } from '$lib/motion';
  import { modal } from '$lib/modal';
  import Icon from './Icon.svelte';

  let { title, subtitle = '', size = 'md', closeLabel = 'Close', onclose, children, footer }: {
    title: string;
    subtitle?: string;
    size?: 'sm' | 'md' | 'lg';
    closeLabel?: string;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
  } = $props();

  const uid = `m${Math.random().toString(36).slice(2, 8)}`;
  const phone = typeof matchMedia !== 'undefined' && matchMedia('(max-width: 640px)').matches;

  // The page behind a dialog does not scroll; stacked dialogs share one lock.
  $effect(() => {
    const n = Number(document.body.dataset.modals ?? 0) + 1;
    document.body.dataset.modals = String(n);
    document.body.style.overflow = 'hidden';
    return () => {
      const left = Number(document.body.dataset.modals ?? 1) - 1;
      document.body.dataset.modals = String(left);
      if (left <= 0) document.body.style.overflow = '';
    };
  });

  let body: HTMLElement;
  let top = $state(false);
  let bottom = $state(false);
  function measure() {
    if (!body) return;
    top = body.scrollTop > 2;
    bottom = body.scrollTop + body.clientHeight < body.scrollHeight - 2;
  }
  $effect(() => { measure(); });
</script>

<div class="backdrop" transition:fade|global={{ duration: ms(180) }} role="presentation" onclick={(e) => e.target === e.currentTarget && onclose()}>
  <div class="sheet {size}" role="dialog" aria-modal="true" aria-labelledby={uid} use:modal={onclose}
    in:fly|global={{ y: phone ? 80 : 20, duration: ms(300), easing: ease }} out:fly|global={{ y: phone ? 80 : 10, duration: ms(170) }}>
    <span class="grip" aria-hidden="true"></span>
    <header class:line={top}>
      <div class="ttl"><h3 id={uid}>{title}</h3>{#if subtitle}<p>{subtitle}</p>{/if}</div>
      <button type="button" class="x" onclick={onclose} aria-label={closeLabel}><Icon name="x" size={18} /></button>
    </header>
    <div class="body" bind:this={body} onscroll={measure}>{@render children()}</div>
    {#if footer}<footer class:line={bottom}>{@render footer()}</footer>{/if}
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; z-index: 80; display: grid; place-items: center; padding: 16px; background: color-mix(in srgb, #06131a 62%, transparent); backdrop-filter: blur(5px); }
  .sheet { position: relative; width: 100%; max-height: min(92vh, 860px); display: flex; flex-direction: column; background: var(--surface); border-radius: 26px; box-shadow: 0 30px 80px #0006, 0 0 0 1px var(--line); overflow: hidden; }
  .sm { max-width: 420px; } .md { max-width: 560px; } .lg { max-width: 760px; }
  .grip { display: none; }
  header { display: flex; align-items: flex-start; gap: .8rem; padding: 1.15rem 1.3rem .9rem; border-bottom: 1px solid transparent; transition: border-color .2s; }
  header.line { border-color: var(--line); }
  .ttl { flex: 1; min-width: 0; } h3 { margin: 0; font-size: 1.15rem; letter-spacing: -.01em; } p { margin: .2rem 0 0; font-size: .85rem; color: var(--muted); }
  .x { flex: none; width: 36px; height: 36px; padding: 0; display: grid; place-items: center; border: 0; background: var(--surface-2); color: var(--muted); border-radius: 50%; min-height: 0; }
  .x:hover { color: var(--ink); background: var(--line); }
  .body { padding: .4rem 1.3rem 1.2rem; overflow-y: auto; overscroll-behavior: contain; flex: 1; }
  footer { display: flex; gap: .6rem; align-items: center; justify-content: flex-end; flex-wrap: wrap; padding: .9rem 1.3rem 1.1rem; border-top: 1px solid transparent; background: var(--surface); transition: border-color .2s; }
  footer.line { border-color: var(--line); }
  @media (max-width: 640px) {
    .backdrop { place-items: end center; padding: 0; }
    .sheet { max-width: none; border-radius: 26px 26px 0 0; max-height: 92vh; padding-bottom: env(safe-area-inset-bottom); }
    .grip { display: block; width: 42px; height: 5px; border-radius: 99px; background: var(--line); margin: .55rem auto 0; }
    header { padding-top: .6rem; }
  }
</style>

<script lang="ts">
  import Icon from './Icon.svelte';
  let { icon = 'note', title, text = '' }: { icon?: string; title: string; text?: string } = $props();
</script>

<div class="empty">
  <span class="ic"><Icon name={icon} size={22} /></span>
  <strong>{title}</strong>
  {#if text}<span class="muted">{text}</span>{/if}
</div>

<style>
  .empty { display: grid; justify-items: center; gap: .35rem; text-align: center; padding: 1.8rem 1rem; }
  .ic { width: 48px; height: 48px; border-radius: 50%; background: var(--surface-2); color: var(--accent); display: grid; place-items: center; margin-bottom: .2rem; }
  .muted { font-size: .85rem; max-width: 42ch; }
</style>

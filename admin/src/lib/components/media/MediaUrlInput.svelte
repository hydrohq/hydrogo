<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import MediaPickerModal from './MediaPickerModal.svelte';

  // A URL field that is not URL-only: pick from the media library, upload a new file, or still type/paste an address.
  let { value = $bindable(''), id, placeholder = '' }: { value?: string; id?: string; placeholder?: string } = $props();
  let open = $state(false);
  const src = $derived(value && /^(https?:)?\/\/|^\/|^data:/.test(value) ? value : value ? `/media/${value}` : '');
</script>

<div class="flex w-full flex-wrap items-center gap-2">
  {#if src}
    <div class="size-10 shrink-0 overflow-hidden rounded-md border border-border bg-muted">
      <img {src} alt="" class="size-full object-cover" onerror={(e) => ((e.currentTarget as HTMLImageElement).style.display = 'none')} />
    </div>
  {/if}
  <input {id} type="text" bind:value {placeholder} class="h-9 min-w-0 flex-1 basis-40 rounded-md border border-input bg-background px-3 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs" />
  <button type="button" onclick={() => (open = true)} class="h-9 shrink-0 rounded-md border border-input bg-background px-3 text-xs font-medium text-foreground shadow-xs transition-colors hover:bg-accent cursor-pointer">{t('products.editor.browse')} / Yükle</button>
  {#if value}
    <button type="button" aria-label="Temizle" onclick={() => (value = '')} class="h-9 shrink-0 rounded-md border border-input bg-background px-2.5 text-xs text-muted-foreground shadow-xs transition-colors hover:bg-destructive/10 hover:text-destructive cursor-pointer">✕</button>
  {/if}
</div>

<MediaPickerModal
  bind:open
  onclose={() => (open = false)}
  onselect={(m: { url?: string; path?: string; id: string }) => {
    value = m.url || m.path || `/media/${m.id}`;
    open = false;
  }}
/>

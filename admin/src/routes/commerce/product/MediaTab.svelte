<script lang="ts">
  import { api, ApiClientError } from '$lib/api/client';
  import { confirmAction } from '$lib/confirm';
  import { t } from '$lib/i18n/index.svelte';
  import PageSpinner from '$lib/components/ui/PageSpinner.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import MediaPickerModal from '$lib/components/media/MediaPickerModal.svelte';
  import type { ProductRecord } from '../ProductEditorView.svelte';

  // Product photos and videos: several per product, and each colour can have its own cover and its own set.
  let { product }: { product: ProductRecord | null } = $props();

  interface Item { id: string; kind?: string; image?: string; video_url?: string; poster?: string; sort_order?: number; variant_color?: string; is_cover?: boolean }
  const API = '/api/v1/admin/content/product_images';
  const productId = $derived(product && product.id && product.id !== 'new' ? product.id : '');

  let items = $state<Item[]>([]);
  let colors = $state<string[]>([]);
  let scope = $state(''); // '' = every colour
  let loading = $state(true);
  let busy = $state(false);
  let error = $state('');
  let pickerOpen = $state(false);
  let posterFor = $state<Item | null>(null);
  let fileInput: HTMLInputElement | null = $state(null);

  const normColor = (c?: string) => (c ?? '').trim();
  const scopeItems = $derived(
    items
      .filter((i) => normColor(i.variant_color).toLocaleLowerCase() === scope.toLocaleLowerCase())
      .sort((a, b) => Number(Boolean(b.is_cover)) - Number(Boolean(a.is_cover)) || (a.sort_order ?? 0) - (b.sort_order ?? 0))
  );
  const url = (v?: string) => (!v ? '' : /^(https?:)?\/\/|^\//.test(v) ? v : `/media/${v}`);
  const isVideo = (i: Item) => i.kind === 'video' && !!i.video_url;
  const msg = (e: unknown, fallback: string) => (e instanceof ApiClientError ? e.message : fallback);

  async function load() {
    if (!productId) { loading = false; return; }
    loading = true;
    try {
      const [mediaRes, variantRes] = await Promise.all([
        api.get<{ data: Item[] }>(API, { params: { 'filter[product]': productId, limit: 100 } }),
        api.get<{ data: { color?: string }[] }>('/api/v1/admin/content/variants', { params: { 'filter[product]': productId, limit: 100 } })
      ]);
      items = mediaRes.data ?? [];
      colors = [...new Set((variantRes.data ?? []).map((v) => normColor(v.color)).filter(Boolean))];
    } catch (e) {
      error = msg(e, 'Media could not be uploaded.');
    } finally {
      loading = false;
    }
  }
  // The product record arrives after the tab mounts; load (again) whenever its id becomes known.
  $effect(() => { if (productId) void load(); else loading = false; });

  const body = (i: Partial<Item>) => ({ product: productId, kind: i.kind ?? 'image', image: i.image ?? '', video_url: i.video_url ?? '', poster: i.poster ?? '', sort_order: i.sort_order ?? 0, variant_color: normColor(i.variant_color), is_cover: Boolean(i.is_cover) });
  const nextOrder = () => Math.max(-1, ...scopeItems.map((i) => i.sort_order ?? 0)) + 1;

  async function addRecord(asset: { url?: string; path?: string; id: string; mime_type?: string; content_type?: string }) {
    const src = asset.url || asset.path || `/media/${asset.id}`;
    const mime = asset.mime_type || asset.content_type || '';
    const video = mime.startsWith('video/') || /\.(mp4|webm|mov)(\?|$)/i.test(src);
    await api.post(API, body({ kind: video ? 'video' : 'image', image: video ? '' : src, video_url: video ? src : '', variant_color: scope, sort_order: nextOrder() }));
  }

  async function uploadFiles(e: Event) {
    const input = e.target as HTMLInputElement;
    const files = [...(input.files ?? [])];
    input.value = '';
    if (!files.length) return;
    busy = true;
    error = '';
    try {
      for (const file of files) {
        const form = new FormData();
        form.append('file', file);
        form.append('visibility', 'public');
        form.append('ready', 'true');
        const res = await api.upload<{ data: { url?: string; path?: string; id: string; asset_id?: string; mime_type?: string; content_type?: string } }>('/api/v1/admin/media', form);
        await addRecord({ ...res.data, id: res.data.id || res.data.asset_id || '' });
      }
      await load();
    } catch (err) {
      error = msg(err, t('commerce.media.uploadFailed'));
      await load();
    } finally {
      busy = false;
    }
  }

  async function pickedFromLibrary(asset: { url?: string; path?: string; id: string; mime_type?: string; content_type?: string }) {
    pickerOpen = false;
    busy = true;
    error = '';
    try {
      if (posterFor) {
        await api.put(`${API}/${posterFor.id}`, body({ ...posterFor, poster: asset.url || asset.path || `/media/${asset.id}` }));
        posterFor = null;
      } else {
        await addRecord(asset);
      }
      await load();
    } catch (err) {
      error = msg(err, 'Kaydedilemedi.');
    } finally {
      busy = false;
    }
  }

  async function toggleCover(item: Item) {
    busy = true;
    error = '';
    try {
      const turnOn = !item.is_cover;
      for (const other of scopeItems.filter((i) => i.is_cover && i.id !== item.id)) {
        await api.put(`${API}/${other.id}`, body({ ...other, is_cover: false }));
      }
      await api.put(`${API}/${item.id}`, body({ ...item, is_cover: turnOn }));
      await load();
    } catch (err) {
      error = msg(err, 'The cover could not be changed.');
    } finally {
      busy = false;
    }
  }

  async function move(item: Item, dir: -1 | 1) {
    const list = scopeItems;
    const at = list.findIndex((i) => i.id === item.id);
    const other = list[at + dir];
    if (!other) return;
    busy = true;
    try {
      // Renumber the whole scope so equal or missing sort_order values never block a move.
      const order = list.map((i) => i.id);
      [order[at], order[at + dir]] = [order[at + dir], order[at]];
      for (const [index, id] of order.entries()) {
        const current = list.find((i) => i.id === id)!;
        if ((current.sort_order ?? -1) !== index) await api.put(`${API}/${id}`, body({ ...current, sort_order: index }));
      }
      await load();
    } catch (err) {
      error = msg(err, 'The order could not be changed.');
    } finally {
      busy = false;
    }
  }

  async function remove(item: Item) {
    const ok = await confirmAction({
      title: t('commerce.media.deleteTitle'),
      description: t('commerce.media.deleteDesc'),
      confirmLabel: t('commerce.media.deleteConfirm'),
      cancelLabel: t('commerce.media.deleteCancel'),
      variant: 'destructive'
    });
    if (!ok) return;
    busy = true;
    try {
      await api.delete(`${API}/${item.id}`);
      await load();
    } catch (err) {
      error = msg(err, 'Silinemedi.');
    } finally {
      busy = false;
    }
  }
</script>

{#if !productId}
  <p class="rounded-lg border border-border bg-card p-6 text-sm text-muted-foreground">{t('commerce.media.saveFirst')}</p>
{:else if loading}
  <PageSpinner />
{:else}
  <div class="space-y-5">
    <div>
      <h2 class="text-sm font-semibold">{t('commerce.media.title')}</h2>
      <p class="mt-1 text-xs text-muted-foreground">{t('commerce.media.help')}</p>
    </div>

    <div class="flex flex-wrap gap-2" role="tablist" aria-label="Renk">
      {#each ['', ...colors] as c (c)}
        <button type="button" role="tab" aria-selected={scope === c} onclick={() => (scope = c)} class="h-8 rounded-full border px-3 text-xs font-medium transition-colors cursor-pointer {scope === c ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-muted-foreground hover:text-foreground'}">
          {c || t('commerce.media.allColors')}<span class="ml-1.5 opacity-70">{items.filter((i) => normColor(i.variant_color).toLocaleLowerCase() === c.toLocaleLowerCase()).length}</span>
        </button>
      {/each}
    </div>
    {#if colors.length === 0}<p class="text-xs text-muted-foreground">{t('commerce.media.colorsHint')}</p>{/if}

    <div class="flex flex-wrap gap-2">
      <input bind:this={fileInput} type="file" multiple accept="image/*,video/mp4,video/webm,video/quicktime" class="hidden" onchange={uploadFiles} />
      <Button size="sm" loading={busy} onclick={() => fileInput?.click()}>{t('commerce.media.upload')}</Button>
      <Button size="sm" variant="outline" disabled={busy} onclick={() => { posterFor = null; pickerOpen = true; }}>{t('commerce.media.library')}</Button>
    </div>
    {#if error}<p role="alert" class="text-xs text-destructive">{error}</p>{/if}

    {#if scopeItems.length === 0}
      <p class="rounded-lg border border-dashed border-border p-8 text-center text-xs text-muted-foreground">{scope ? t('commerce.media.emptyColor', { color: scope }) : t('commerce.media.empty')}</p>
    {:else}
      <ul class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
        {#each scopeItems as item, i (item.id)}
          <li class="overflow-hidden rounded-lg border border-border bg-card">
            <div class="relative aspect-[4/5] bg-muted">
              {#if isVideo(item)}
                {#if item.poster}<img src={url(item.poster)} alt="" class="size-full object-cover" />{:else}<video src={url(item.video_url)} muted preload="metadata" class="size-full object-cover"></video>{/if}
                <span class="absolute inset-0 flex items-center justify-center"><span class="flex size-9 items-center justify-center rounded-full bg-black/60 text-white"><svg class="ml-0.5 size-4" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></span></span>
              {:else}
                <img src={url(item.image)} alt="" class="size-full object-cover" loading="lazy" />
              {/if}
              {#if item.is_cover}<span class="absolute left-2 top-2 rounded bg-primary px-1.5 py-0.5 text-[10px] font-semibold text-primary-foreground">{t('commerce.media.cover')}</span>{/if}
            </div>
            <div class="flex flex-wrap items-center gap-1 p-2">
              <button type="button" disabled={busy} onclick={() => toggleCover(item)} class="h-7 rounded border border-input px-2 text-[11px] hover:bg-accent cursor-pointer disabled:opacity-50">{item.is_cover ? t('commerce.media.removeCover') : t('commerce.media.makeCover')}</button>
              <button type="button" disabled={busy || i === 0} aria-label={t('commerce.media.moveEarlier')} onclick={() => move(item, -1)} class="h-7 w-7 rounded border border-input text-xs hover:bg-accent cursor-pointer disabled:opacity-40">←</button>
              <button type="button" disabled={busy || i === scopeItems.length - 1} aria-label={t('commerce.media.moveLater')} onclick={() => move(item, 1)} class="h-7 w-7 rounded border border-input text-xs hover:bg-accent cursor-pointer disabled:opacity-40">→</button>
              {#if isVideo(item)}<button type="button" disabled={busy} onclick={() => { posterFor = item; pickerOpen = true; }} class="h-7 rounded border border-input px-2 text-[11px] hover:bg-accent cursor-pointer">{t('commerce.media.videoCover')}</button>{/if}
              <button type="button" disabled={busy} aria-label={t('commerce.media.deleteConfirm')} onclick={() => remove(item)} class="ml-auto h-7 w-7 rounded border border-input text-xs text-muted-foreground hover:bg-destructive/10 hover:text-destructive cursor-pointer">✕</button>
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </div>

  <MediaPickerModal bind:open={pickerOpen} onclose={() => { pickerOpen = false; posterFor = null; }} onselect={pickedFromLibrary} />
{/if}

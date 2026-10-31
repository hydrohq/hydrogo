<script lang="ts">
  // Code-splits a route view: the chunk for `load` downloads only once this
  // component is actually rendered, instead of being bundled into the app shell.
  import ContentSkeleton, { type SkeletonKind } from './ContentSkeleton.svelte';
  import { router } from '$lib/router.svelte';

  type Props = {
    load: () => Promise<{ default: unknown }>;
    props?: Record<string, unknown>;
    skeletonKind?: SkeletonKind;
  };

  let { load, props = {}, skeletonKind }: Props = $props();

  const kind = $derived.by<SkeletonKind>(() => {
    if (skeletonKind) return skeletonKind;
    const p = (router.path || '').toLowerCase();
    const isConsole = typeof window !== 'undefined' && (
      window.location.hostname.startsWith('console.') ||
      window.location.hostname.includes('console.example.com') ||
      p.startsWith('/console') ||
      p.startsWith('/orchestrator')
    );
    if (isConsole) return 'console';
    if (p === '/' || p === '/dashboard') return 'dashboard';
    if (p.startsWith('/media')) return 'media';
    if (p.startsWith('/site') || p.startsWith('/account')) return 'settings';
    if (
      (p.startsWith('/pages/') && p !== '/pages') ||
      (p.startsWith('/news/') && p !== '/news') ||
      p.includes('/new') ||
      p.includes('/edit')
    ) {
      return 'editor';
    }
    return 'table';
  });
</script>

{#await load()}
  <ContentSkeleton {kind} />
{:then mod}
  {@const Component = mod.default as any}
  <Component {...props} />
{:catch error}
  <div class="p-8 max-w-2xl mx-auto rounded-xl border border-destructive/30 bg-destructive/5 text-center space-y-2">
    <p class="text-sm font-medium text-destructive">Failed to load view</p>
    <p class="text-xs text-muted-foreground">{error instanceof Error ? error.message : String(error)}</p>
  </div>
{/await}

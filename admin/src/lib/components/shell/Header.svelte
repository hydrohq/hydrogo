<script lang="ts">
  import { router } from '$lib/router.svelte';
  import { schema } from '$lib/stores/schema.svelte';

  const path = $derived(router.path);

  const title = $derived.by(() => {
    if (path === '/' || path === '/dashboard') return 'Dashboard';
    if (path.startsWith('/content/')) {
      const parts = path.split('/');
      const type = parts[2];
      const collection = schema.getCollection(type);
      if (collection) {
        if (parts[3] === 'new') return `New ${collection.singular || collection.label}`;
        if (parts[3]) return `Edit ${collection.singular || collection.label}`;
        return collection.label || collection.plural;
      }
      return 'Content';
    }
    if (path.startsWith('/pages')) return 'Pages';
    if (path.startsWith('/news')) return 'News';
    if (path.startsWith('/media')) return 'Media Library';
    if (path.startsWith('/members')) return 'Members';
    if (path.startsWith('/site')) return 'Site Settings';
    if (path.startsWith('/users')) return 'Users & Access';
    if (path.startsWith('/audit')) return 'Audit Trail';
    if (path.startsWith('/system')) return 'System Metrics';
    return 'Admin';
  });
</script>

<header class="h-16 px-8 border-b border-slate-800/80 bg-slate-950/80 backdrop-blur flex items-center justify-between flex-shrink-0 z-10">
  <div class="flex items-center gap-3">
    <h1 class="text-lg font-bold text-slate-100 tracking-tight">{title}</h1>
  </div>

  <div class="flex items-center gap-4">
    <!-- Locale indicator -->
    {#if schema.locales.length > 1}
      <div class="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-900 border border-slate-800 px-2.5 py-1 rounded-lg">
        <span class="text-slate-500">Locale:</span>
        <span class="font-semibold text-sky-400 uppercase">{schema.defaultLocale}</span>
      </div>
    {/if}

    <a
      href="/"
      target="_blank"
      class="text-xs text-slate-400 hover:text-sky-400 flex items-center gap-1.5 transition-colors px-2.5 py-1 rounded-lg hover:bg-slate-900"
    >
      <span>Live Site</span>
      <span>↗</span>
    </a>
  </div>
</header>

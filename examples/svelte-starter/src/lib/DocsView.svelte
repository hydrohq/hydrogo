<script lang="ts">
  import { docCategories, docTopics, type DocTopic } from './docsData';
  import { setDocsSEO } from './seo';

  interface Props {
    onback: () => void;
  }

  let { onback }: Props = $props();

  let activeTopicId = $state<string>('overview');
  let searchQuery = $state('');
  let mobileSidebarOpen = $state(false);
  let copiedCodeId = $state<string | null>(null);

  const currentTopic = $derived<DocTopic>(docTopics[activeTopicId] || docTopics['overview']);

  $effect(() => {
    if (currentTopic) {
      setDocsSEO(currentTopic.title);
    }
  });

  // Flattened topics list for prev/next pagination
  const allTopicIds = $derived(
    docCategories.flatMap((cat) => cat.topics.map((t) => t.id))
  );

  const currentIndex = $derived(allTopicIds.indexOf(activeTopicId));
  const prevTopic = $derived(currentIndex > 0 ? docTopics[allTopicIds[currentIndex - 1]] : null);
  const nextTopic = $derived(
    currentIndex >= 0 && currentIndex < allTopicIds.length - 1
      ? docTopics[allTopicIds[currentIndex + 1]]
      : null
  );

  // Search filtering
  const filteredCategories = $derived.by(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return docCategories;
    return docCategories
      .map((cat) => ({
        ...cat,
        topics: cat.topics.filter(
          (t) =>
            t.title.toLowerCase().includes(q) ||
            (docTopics[t.id]?.description || '').toLowerCase().includes(q) ||
            (docTopics[t.id]?.headings || []).some((h) => h.toLowerCase().includes(q))
        ),
      }))
      .filter((cat) => cat.topics.length > 0);
  });

  function selectTopic(id: string) {
    activeTopicId = id;
    mobileSidebarOpen = false;
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  async function copyCode(code: string, id: string) {
    try {
      await navigator.clipboard.writeText(code);
      copiedCodeId = id;
      setTimeout(() => {
        if (copiedCodeId === id) copiedCodeId = null;
      }, 2000);
    } catch (e) {
      console.warn('Copy failed', e);
    }
  }
</script>

<div class="min-h-screen bg-black text-zinc-100 flex flex-col font-sans antialiased selection:bg-white selection:text-black">
  <!-- Top Global Documentation Navbar -->
  <header class="sticky top-0 z-40 border-b border-zinc-800/80 bg-black/85 backdrop-blur-md">
    <div class="w-full max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-4">
      <!-- Left: Logo & Docs Brand -->
      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => (mobileSidebarOpen = !mobileSidebarOpen)}
          class="p-1.5 rounded-lg border border-zinc-800 text-zinc-400 hover:text-white hover:bg-zinc-900 lg:hidden cursor-pointer"
          aria-label="Toggle navigation menu"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"></line>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <line x1="3" y1="18" x2="21" y2="18"></line>
          </svg>
        </button>

        <button
          type="button"
          onclick={onback}
          class="flex items-center gap-2.5 text-white font-bold text-lg hover:opacity-90 transition-opacity cursor-pointer bg-transparent border-0 p-0"
        >
          <div class="size-7 rounded-lg bg-white/10 border border-white/15 flex items-center justify-center text-white">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
              <polyline points="2 17 12 22 22 17"></polyline>
              <polyline points="2 12 12 17 22 12"></polyline>
            </svg>
          </div>
          <span>Hydro<span class="text-sky-400">Go</span></span>
        </button>

        <span class="inline-flex items-center px-2 py-0.5 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-400 text-xs font-mono font-medium">
          Docs
        </span>

        <span class="hidden sm:inline-block text-xs font-mono text-zinc-500 bg-zinc-900 px-2 py-0.5 rounded border border-zinc-800">
          v2.0
        </span>
      </div>

      <!-- Center: Search Input -->
      <div class="flex-1 max-w-md hidden md:block">
        <div class="relative">
          <svg class="size-4 absolute left-3 top-2.5 text-zinc-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            type="text"
            bind:value={searchQuery}
            placeholder="Search documentation, API endpoints, YAML..."
            class="w-full h-9 pl-9 pr-12 rounded-full bg-zinc-900 border border-zinc-800 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-zinc-600 focus:ring-1 focus:ring-zinc-600 transition-colors"
          />
          <kbd class="absolute right-3 top-2 text-[10px] font-mono text-zinc-500 bg-zinc-800 px-1.5 py-0.5 rounded border border-zinc-700">⌘K</kbd>
        </div>
      </div>

      <!-- Right: Action Links -->
      <div class="flex items-center gap-3 text-xs font-medium">
        <!-- Exit to Starter Button -->
        <button
          type="button"
          onclick={onback}
          class="h-8 px-3.5 rounded-full bg-white text-black hover:bg-zinc-200 font-semibold transition-colors flex items-center gap-1.5 cursor-pointer shadow-xs"
        >
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="19" y1="12" x2="5" y2="12"></line>
            <polyline points="12 19 5 12 12 5"></polyline>
          </svg>
          Exit Docs
        </button>
      </div>
    </div>
  </header>

  <!-- Mobile Search Bar (Only on small screens) -->
  <div class="p-3 border-b border-zinc-800 bg-zinc-950 md:hidden">
    <div class="relative">
      <svg class="size-4 absolute left-3 top-2.5 text-zinc-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Search docs..."
        class="w-full h-9 pl-9 pr-4 rounded-lg bg-zinc-900 border border-zinc-800 text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none"
      />
    </div>
  </div>

  <!-- Main Body: 3-Column Layout -->
  <div class="w-full max-w-7xl mx-auto flex-1 flex">
    <!-- Left Navigation Sidebar -->
    <aside
      class="fixed inset-y-0 left-0 z-50 w-72 bg-black border-r border-zinc-800/80 p-6 overflow-y-auto transition-transform duration-200 ease-in-out lg:static lg:w-64 lg:p-6 lg:translate-x-0 {mobileSidebarOpen ? 'translate-x-0' : '-translate-x-full'}"
    >
      <!-- Mobile Close Button -->
      <div class="flex items-center justify-between pb-4 border-b border-zinc-800 lg:hidden mb-4">
        <span class="text-xs font-semibold uppercase tracking-wider text-zinc-400">Documentation</span>
        <button
          type="button"
          onclick={() => (mobileSidebarOpen = false)}
          class="p-1 rounded text-zinc-400 hover:text-white"
        >
          ✕
        </button>
      </div>

      <nav class="space-y-6">
        {#each filteredCategories as category}
          <div class="space-y-1.5">
            <h3 class="text-xs font-semibold uppercase tracking-wider text-zinc-400 px-2 font-mono">
              {category.title}
            </h3>
            <ul class="space-y-0.5">
              {#each category.topics as topic}
                <li>
                  <button
                    type="button"
                    onclick={() => selectTopic(topic.id)}
                    class="w-full text-left px-2.5 py-1.5 rounded-lg text-xs font-medium transition-colors flex items-center justify-between cursor-pointer {activeTopicId === topic.id ? 'bg-white/10 text-white font-semibold' : 'text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900/60'}"
                  >
                    <span class="truncate">{topic.title}</span>
                    {#if topic.badge}
                      <span class="ml-2 text-[10px] font-mono px-1.5 py-0.2 rounded bg-sky-500/10 text-sky-400 border border-sky-500/20">
                        {topic.badge}
                      </span>
                    {/if}
                  </button>
                </li>
              {/each}
            </ul>
          </div>
        {/each}
      </nav>
    </aside>

    <!-- Backdrop for mobile sidebar -->
    {#if mobileSidebarOpen}
      <button
        type="button"
        onclick={() => (mobileSidebarOpen = false)}
        class="fixed inset-0 z-40 bg-black/70 backdrop-blur-xs lg:hidden"
        aria-label="Close sidebar"
      ></button>
    {/if}

    <!-- Central Reading Article Pane -->
    <main class="flex-1 min-w-0 px-6 sm:px-10 py-10 max-w-4xl mx-auto">
      <!-- Breadcrumb -->
      <div class="flex items-center gap-2 text-xs text-zinc-500 font-mono mb-4">
        <span>Docs</span>
        <span>/</span>
        <span>{currentTopic.category}</span>
        <span>/</span>
        <span class="text-zinc-300 font-medium">{currentTopic.title}</span>
      </div>

      <!-- Article Title Header -->
      <div class="space-y-3 pb-8 border-b border-zinc-800">
        <div class="flex flex-wrap items-center gap-2.5">
          <h1 class="text-3xl sm:text-4xl font-extrabold text-white tracking-tight">
            {currentTopic.title}
          </h1>
          {#if currentTopic.badge}
            <span class="text-xs font-mono px-2 py-0.5 rounded-full bg-sky-500/10 text-sky-400 border border-sky-500/20">
              {currentTopic.badge}
            </span>
          {/if}
        </div>
        <p class="text-base text-zinc-400 leading-relaxed">
          {currentTopic.description}
        </p>
      </div>

      <!-- Lead paragraph -->
      <div class="py-6 text-sm text-zinc-300 leading-relaxed font-normal">
        {currentTopic.content.lead}
      </div>

      <!-- Article Subsections -->
      <div class="space-y-10">
        {#each currentTopic.content.sections as section, sIdx}
          <section id={section.id} class="space-y-4 pt-2">
            <h2 class="text-xl sm:text-2xl font-bold text-white tracking-tight flex items-center gap-2 group">
              <a href="#{section.id}" class="text-zinc-600 group-hover:text-sky-400 font-mono text-base transition-colors">#</a>
              <span>{section.title}</span>
            </h2>

            <p class="text-sm text-zinc-300 leading-relaxed">
              {section.text}
            </p>

            <!-- Callout Alerts -->
            {#if section.callout}
              <div
                class="p-4 rounded-xl border text-xs leading-relaxed flex items-start gap-3 {section.callout.type === 'tip' ? 'bg-sky-500/5 border-sky-500/20 text-sky-200' : section.callout.type === 'warning' ? 'bg-amber-500/5 border-amber-500/20 text-amber-200' : 'bg-zinc-900 border-zinc-800 text-zinc-300'}"
              >
                <div class="size-5 rounded-md flex items-center justify-center shrink-0 mt-0.5 {section.callout.type === 'tip' ? 'text-sky-400' : section.callout.type === 'warning' ? 'text-amber-400' : 'text-zinc-400'}">
                  {#if section.callout.type === 'tip'}
                    ðŸ’¡
                  {:else if section.callout.type === 'warning'}
                    ⚠️
                  {:else}
                    ℹ️
                  {/if}
                </div>
                <div>{section.callout.text}</div>
              </div>
            {/if}

            <!-- Code Blocks with Filename Tab and Copy Button -->
            {#if section.code}
              {@const codeId = `${currentTopic.id}-${sIdx}`}
              <div class="rounded-xl border border-zinc-800 bg-zinc-950 overflow-hidden shadow-xs">
                <div class="flex items-center justify-between px-4 py-2 bg-zinc-900/70 border-b border-zinc-800/80 text-xs text-zinc-400 font-mono">
                  <span>{section.code.filename || section.code.lang}</span>
                  <button
                    type="button"
                    onclick={() => copyCode(section.code!.code, codeId)}
                    class="hover:text-white transition-colors flex items-center gap-1 cursor-pointer bg-transparent border-0 p-0 text-zinc-400 text-xs"
                  >
                    {#if copiedCodeId === codeId}
                      <span class="text-emerald-400">✓ Copied</span>
                    {:else}
                      <span>Copy</span>
                    {/if}
                  </button>
                </div>
                <div class="p-4 font-mono text-xs text-zinc-200 overflow-x-auto leading-relaxed">
                  <pre><code>{section.code.code}</code></pre>
                </div>
              </div>
            {/if}

            <!-- Structured Tables -->
            {#if section.table}
              <div class="rounded-xl border border-zinc-800 overflow-hidden text-xs">
                <table class="w-full text-left divide-y divide-zinc-800">
                  <thead class="bg-zinc-900/60 font-mono text-zinc-400 uppercase">
                    <tr>
                      {#each section.table.headers as head}
                        <th class="p-3 px-4 font-semibold">{head}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-zinc-800/60 bg-zinc-950/40">
                    {#each section.table.rows as row}
                      <tr class="hover:bg-zinc-900/40 transition-colors">
                        {#each row as cell, cIdx}
                          <td class="p-3 px-4 {cIdx === 0 ? 'font-mono text-sky-400 font-medium' : 'text-zinc-300'}">
                            {cell}
                          </td>
                        {/each}
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            {/if}
          </section>
        {/each}
      </div>

      <!-- Prev / Next Pagination Cards -->
      <div class="mt-16 pt-8 border-t border-zinc-800/80 grid grid-cols-1 sm:grid-cols-2 gap-4">
        {#if prevTopic}
          <button
            type="button"
            onclick={() => selectTopic(prevTopic.id)}
            class="p-4 rounded-xl border border-zinc-800 bg-zinc-950 hover:bg-zinc-900/70 hover:border-zinc-700 transition-colors text-left space-y-1 cursor-pointer"
          >
            <span class="text-[10px] font-mono text-zinc-500 uppercase tracking-wider">← Previous</span>
            <div class="text-sm font-semibold text-white">{prevTopic.title}</div>
            <p class="text-xs text-zinc-400 line-clamp-1">{prevTopic.description}</p>
          </button>
        {:else}
          <div></div>
        {/if}

        {#if nextTopic}
          <button
            type="button"
            onclick={() => selectTopic(nextTopic.id)}
            class="p-4 rounded-xl border border-zinc-800 bg-zinc-950 hover:bg-zinc-900/70 hover:border-zinc-700 transition-colors text-right space-y-1 cursor-pointer sm:col-start-2"
          >
            <span class="text-[10px] font-mono text-zinc-500 uppercase tracking-wider">Next →</span>
            <div class="text-sm font-semibold text-white">{nextTopic.title}</div>
            <p class="text-xs text-zinc-400 line-clamp-1">{nextTopic.description}</p>
          </button>
        {/if}
      </div>
    </main>
  </div>
</div>

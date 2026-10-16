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
  const currentSections = $derived(currentTopic.content.sections);

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
            (docTopics[t.id]?.headings || []).some((h) => h.toLowerCase().includes(q)) ||
            (docTopics[t.id]?.content.lead || '').toLowerCase().includes(q) ||
            (docTopics[t.id]?.content.sections || []).some((section) =>
              `${section.title} ${section.text} ${section.code?.code || ''}`.toLowerCase().includes(q)
            )
        ),
      }))
      .filter((cat) => cat.topics.length > 0);
  });

  function selectTopic(id: string) {
    activeTopicId = id;
    mobileSidebarOpen = false;
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function handleSearchKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') searchQuery = '';
    if (event.key === 'Enter') {
      const firstTopic = filteredCategories[0]?.topics[0];
      if (firstTopic) selectTopic(firstTopic.id);
    }
  }

  function openSearchShortcut(event: KeyboardEvent) {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      document.querySelector<HTMLInputElement>('[data-docs-search]')?.focus();
    }
  }

  $effect(() => {
    window.addEventListener('keydown', openSearchShortcut);
    return () => window.removeEventListener('keydown', openSearchShortcut);
  });

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

<div class="docs-shell min-h-screen flex flex-col font-sans antialiased selection:bg-[#dedad1] selection:text-[#202321]">
  <!-- Top Global Documentation Navbar -->
  <header class="docs-header sticky top-0 z-40 border-b">
    <div class="w-full max-w-[1480px] mx-auto px-4 sm:px-6 h-[68px] flex items-center justify-between gap-4">
      <!-- Left: Logo & Docs Brand -->
      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => (mobileSidebarOpen = !mobileSidebarOpen)}
          class="docs-menu p-2 border lg:hidden cursor-pointer"
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
          class="flex items-center gap-2.5 docs-wordmark font-bold text-lg hover:opacity-75 transition-opacity cursor-pointer bg-transparent border-0 p-0"
        >
          <div class="size-7 border flex items-center justify-center">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M12 3 3 8l9 5 9-5-9-5Z"/><path d="m3 12 9 5 9-5M3 16l9 5 9-5"/>
            </svg>
          </div>
          <span>Hydro<span class="docs-accent">Go</span></span>
        </button>

        <span class="docs-label inline-flex items-center px-2 py-0.5 text-xs font-mono font-medium">
          Docs
        </span>

        <span class="hidden sm:inline-block text-xs font-mono docs-muted px-2 py-0.5 border">
          v2.0
        </span>
      </div>

      <!-- Center: Search Input -->
      <div class="flex-1 max-w-[440px] hidden md:block">
        <div class="relative">
          <svg class="size-4 absolute left-3 top-2.5 docs-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            type="text"
            data-docs-search
            bind:value={searchQuery}
            onkeydown={handleSearchKeydown}
            placeholder="Search the documentation..."
            class="docs-search w-full h-10 pl-10 pr-12 border text-sm focus:outline-none transition-colors"
          />
          <kbd class="absolute right-3 top-2.5 text-[10px] font-mono docs-muted px-1.5 py-0.5 border">⌘ K</kbd>
        </div>
      </div>

      <!-- Right: Action Links -->
      <div class="flex items-center gap-3 text-xs font-medium">
        <!-- Exit to Starter Button -->
        <button
          type="button"
          onclick={onback}
          class="docs-exit h-9 px-3.5 font-semibold transition-colors flex items-center gap-2 cursor-pointer"
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
  <div class="docs-mobile-search p-3 border-b md:hidden">
    <div class="relative">
      <svg class="size-4 absolute left-3 top-2.5 docs-muted" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
      </svg>
      <input
        type="text"
        data-docs-search
        bind:value={searchQuery}
        onkeydown={handleSearchKeydown}
        placeholder="Search the documentation..."
        class="docs-search w-full h-10 pl-10 pr-4 border text-sm focus:outline-none"
      />
    </div>
  </div>

  <!-- Main Body: 3-Column Layout -->
  <div class="docs-layout w-full max-w-[1480px] mx-auto flex-1 flex">
    <!-- Left Navigation Sidebar -->
    <aside
      class="docs-sidebar fixed inset-y-0 left-0 z-50 w-72 p-6 overflow-y-auto transition-transform duration-200 ease-in-out lg:static lg:w-[270px] lg:p-6 lg:translate-x-0 {mobileSidebarOpen ? 'translate-x-0' : '-translate-x-full'}"
    >
      <!-- Mobile Close Button -->
      <div class="flex items-center justify-between pb-4 border-b lg:hidden mb-4">
        <span class="text-xs font-semibold uppercase tracking-wider docs-muted">Documentation</span>
        <button
          type="button"
          onclick={() => (mobileSidebarOpen = false)}
          class="p-1 docs-muted hover:opacity-70"
        >
          ✕
        </button>
      </div>

      <nav class="space-y-7" aria-label="Documentation topics">
        {#each filteredCategories as category}
          <div class="space-y-1.5">
            <h3 class="text-[10px] font-semibold uppercase tracking-[.15em] docs-muted px-2 font-mono mb-2">
              {category.title}
            </h3>
            <ul class="space-y-0.5">
              {#each category.topics as topic}
                <li>
                  <button
                    type="button"
                    onclick={() => selectTopic(topic.id)}
                    aria-current={activeTopicId === topic.id ? 'page' : undefined}
                    class="docs-topic w-full text-left px-2.5 py-2 text-[13px] transition-colors flex items-center justify-between cursor-pointer {activeTopicId === topic.id ? 'is-active font-semibold' : 'docs-muted hover:opacity-75'}"
                  >
                    <span class="truncate">{topic.title}</span>
                    {#if topic.badge}
                      <span class="docs-label ml-2 text-[9px] font-mono px-1.5 py-0.5 border">
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
        class="fixed inset-0 z-40 bg-black/45 lg:hidden"
        aria-label="Close sidebar"
      ></button>
    {/if}

    <!-- Central Reading Article Pane -->
    <main class="docs-article flex-1 min-w-0 px-6 sm:px-10 xl:px-14 py-10 sm:py-14 max-w-5xl mx-auto">
      <!-- Breadcrumb -->
      <div class="flex items-center gap-2 text-[11px] docs-muted font-mono mb-7">
        <span>Docs</span>
        <span>/</span>
        <span>{currentTopic.category}</span>
        <span>/</span>
        <span class="docs-ink font-medium">{currentTopic.title}</span>
      </div>

      <!-- Article Title Header -->
      <div class="space-y-4 pb-8 border-b">
        <div class="flex flex-wrap items-center gap-2.5">
          <h1 class="text-4xl sm:text-5xl lg:text-[58px] leading-[1.04] font-semibold docs-ink tracking-[-.045em]">
            {currentTopic.title}
          </h1>
          {#if currentTopic.badge}
            <span class="docs-label text-xs font-mono px-2 py-1 border">
              {currentTopic.badge}
            </span>
          {/if}
        </div>
        <p class="text-base sm:text-lg docs-muted leading-relaxed max-w-2xl">
          {currentTopic.description}
        </p>
      </div>

      <!-- Lead paragraph -->
      <div class="py-7 sm:py-8 text-[17px] sm:text-lg docs-body leading-[1.8] font-normal max-w-[70ch]">
        {currentTopic.content.lead}
      </div>

      <!-- Article Subsections -->
      <div class="space-y-12 sm:space-y-14">
        {#each currentTopic.content.sections as section, sIdx}
          <section id={section.id} class="docs-section space-y-4 pt-2 scroll-mt-28">
            <h2 class="text-xl sm:text-2xl font-semibold docs-ink tracking-tight flex items-baseline gap-3 group">
              <a href="#{section.id}" aria-label={`Link to ${section.title}`} class="docs-accent opacity-0 group-hover:opacity-100 focus:opacity-100 font-mono text-sm transition-opacity">#</a>
              <span>{section.title}</span>
            </h2>

            <p class="text-[15px] docs-body leading-[1.85] max-w-[72ch] whitespace-pre-line">
              {section.text}
            </p>

            <!-- Callout Alerts -->
            {#if section.callout}
              <div
                class="docs-callout p-4 border text-sm leading-relaxed flex items-start gap-3 {section.callout.type === 'tip' ? 'is-tip' : section.callout.type === 'warning' ? 'is-warning' : 'is-note'}"
              >
                <div class="size-5 flex items-center justify-center shrink-0 mt-0.5 docs-accent">
                  {#if section.callout.type === 'tip'}
                    ✦
                  {:else if section.callout.type === 'warning'}
                    ⚠️
                  {:else}
                    i
                  {/if}
                </div>
                <div>{section.callout.text}</div>
              </div>
            {/if}

            <!-- Code Blocks with Filename Tab and Copy Button -->
            {#if section.code}
              {@const codeId = `${currentTopic.id}-${sIdx}`}
              <div class="docs-code border overflow-hidden">
                <div class="flex items-center justify-between px-4 py-3 border-b text-xs docs-muted font-mono">
                  <span>{section.code.filename || section.code.lang}</span>
                  <button
                    type="button"
                    onclick={() => copyCode(section.code!.code, codeId)}
                    class="docs-copy hover:opacity-70 transition-colors flex items-center gap-1 cursor-pointer bg-transparent border-0 p-0 text-xs"
                  >
                    {#if copiedCodeId === codeId}
                      <span>✓ Copied</span>
                    {:else}
                      <span>Copy</span>
                    {/if}
                  </button>
                </div>
                <div class="px-4 py-5 font-mono text-[12px] sm:text-[13px] docs-code-text overflow-x-auto leading-[1.8]">
                  <pre><code>{section.code.code}</code></pre>
                </div>
              </div>
            {/if}

            <!-- Structured Tables -->
            {#if section.table}
              <div class="docs-table border overflow-x-auto text-sm">
                <table class="w-full text-left divide-y">
                  <thead class="docs-table-head font-mono docs-muted uppercase">
                    <tr>
                      {#each section.table.headers as head}
                        <th class="p-3 px-4 font-semibold">{head}</th>
                      {/each}
                    </tr>
                  </thead>
                  <tbody class="divide-y">
                    {#each section.table.rows as row}
                      <tr class="docs-table-row transition-colors">
                        {#each row as cell, cIdx}
                          <td class="p-3 px-4 leading-relaxed {cIdx === 0 ? 'font-mono docs-accent font-medium' : 'docs-body'}">
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
      <div class="mt-16 pt-8 border-t grid grid-cols-1 sm:grid-cols-2 gap-3">
        {#if prevTopic}
          <button
            type="button"
            onclick={() => selectTopic(prevTopic.id)}
            class="docs-pagination p-4 border transition-colors text-left space-y-2 cursor-pointer"
          >
            <span class="text-[10px] font-mono docs-muted uppercase tracking-wider">← Previous</span>
            <div class="text-sm font-semibold docs-ink">{prevTopic.title}</div>
            <p class="text-xs docs-muted line-clamp-1">{prevTopic.description}</p>
          </button>
        {:else}
          <div></div>
        {/if}

        {#if nextTopic}
          <button
            type="button"
            onclick={() => selectTopic(nextTopic.id)}
            class="docs-pagination p-4 border transition-colors text-right space-y-2 cursor-pointer sm:col-start-2"
          >
            <span class="text-[10px] font-mono docs-muted uppercase tracking-wider">Next →</span>
            <div class="text-sm font-semibold docs-ink">{nextTopic.title}</div>
            <p class="text-xs docs-muted line-clamp-1">{nextTopic.description}</p>
          </button>
        {/if}
      </div>
    </main>

    <aside class="docs-toc hidden 2xl:block w-[220px] shrink-0 pt-14 pr-6" aria-label="On this page">
      <p class="text-[10px] font-mono docs-muted uppercase tracking-[.15em] mb-4">On this page</p>
      <nav class="space-y-1 border-l pl-4" aria-label="Article sections">
        {#each currentSections as section}
          <a class="docs-toc-link block py-1.5 text-xs docs-muted hover:opacity-70 transition-colors" href="#{section.id}">{section.title}</a>
        {/each}
      </nav>
      <p class="text-[11px] docs-muted leading-relaxed mt-6">Use the topic list to move through the guide, or search across titles and examples.</p>
    </aside>
  </div>
</div>

<style>
  .docs-shell {
    --paper: #f4f2ec;
    --ink: #202321;
    --body: #454944;
    --muted: #777b75;
    --line: #deddd5;
    --accent: #95613f;
    background: var(--paper);
    color: var(--ink);
  }
  .docs-shell :global(button), .docs-shell :global(a), .docs-shell :global(input) { -webkit-tap-highlight-color: transparent; }
  .docs-header, .docs-sidebar { background: var(--paper); border-color: var(--line); }
  .docs-wordmark, .docs-ink { color: var(--ink); }
  .docs-muted { color: var(--muted); }
  .docs-body { color: var(--body); }
  .docs-accent { color: var(--accent); }
  .docs-label { color: var(--accent); border-color: #d8c9bb; background: #eee8df; }
  .docs-menu { color: var(--ink); border-color: var(--line); background: transparent; }
  .docs-search { color: var(--ink); border-color: var(--line); background: #eeece5; }
  .docs-search::placeholder { color: #858880; }
  .docs-search:focus { border-color: #a66a45; box-shadow: 0 0 0 2px #a66a4526; }
  .docs-exit { color: var(--paper); background: var(--ink); }
  .docs-exit:hover { background: #3d413d; }
  .docs-mobile-search { background: #efede7; border-color: var(--line); }
  .docs-topic { color: var(--muted); }
  .docs-topic.is-active { color: var(--ink); background: #e7e4dc; box-shadow: inset 2px 0 #a66a45; }
  .docs-topic:focus-visible, .docs-menu:focus-visible, .docs-exit:focus-visible, .docs-pagination:focus-visible, .docs-copy:focus-visible, .docs-section a:focus-visible, .docs-toc-link:focus-visible { outline: 2px solid var(--accent); outline-offset: 3px; }
  .docs-section > h2 { scroll-margin-top: 6rem; }
  .docs-callout { border-color: #d9d5c9; }
  .docs-callout.is-tip { color: #394a40; background: #e8eee7; border-color: #ccd8cb; }
  .docs-callout.is-warning { color: #6b4a2e; background: #f2e9dc; border-color: #e4d3bb; }
  .docs-callout.is-note { color: var(--body); background: #eeece5; }
  .docs-code { background: #252825; border-color: #252825; }
  .docs-code-text { color: #f1eee7; }
  .docs-copy { color: #c8c7bf; }
  .docs-table { border-color: var(--line); }
  .docs-table-head { background: #eae8e1; }
  .docs-table :global(th), .docs-table :global(td) { border-color: var(--line); }
  .docs-table-row:hover { background: #eeece5; }
  .docs-pagination { background: transparent; border-color: var(--line); }
  .docs-pagination:hover { background: #eeece5; border-color: #c9c6bc; }
  .docs-toc { border-color: var(--line); }
  @media (max-width: 1023px) {
    .docs-sidebar { background: var(--paper); }
  }
  @media (prefers-reduced-motion: reduce) {
    .docs-shell *, .docs-shell *::before, .docs-shell *::after { scroll-behavior: auto !important; transition-duration: .01ms !important; animation-duration: .01ms !important; }
  }
</style>

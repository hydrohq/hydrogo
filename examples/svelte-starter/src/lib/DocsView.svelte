<script lang="ts">
  import { onMount } from 'svelte';
  import { docCategories, docTopics, type DocTopic } from './docsData';
  import BrandLogo from './BrandLogo.svelte';
  import { setDocsSEO } from './seo';

  interface Props {
    onback: () => void;
  }

  let { onback }: Props = $props();

  let activeTopicId = $state<string>('overview');
  let searchQuery = $state('');
  let mobileSidebarOpen = $state(false);
  let copiedCodeId = $state<string | null>(null);
  let theme = $state<'dark' | 'light'>('dark');

  const currentTopic = $derived<DocTopic>(docTopics[activeTopicId] || docTopics['overview']);
  const currentSections = $derived(currentTopic.content.sections);

  onMount(() => {
    const savedTheme = localStorage.getItem('hydrogo-docs-theme');
    if (savedTheme === 'dark' || savedTheme === 'light') theme = savedTheme;
  });

  function toggleTheme() {
    theme = theme === 'dark' ? 'light' : 'dark';
    localStorage.setItem('hydrogo-docs-theme', theme);
  }

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
      Array.from(document.querySelectorAll<HTMLInputElement>('[data-docs-search]'))
        .find((input) => input.getClientRects().length > 0)?.focus();
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

<div class="docs-shell min-h-screen flex flex-col font-sans antialiased" data-theme={theme}>
  <!-- Top Global Documentation Navbar -->
  <header class="docs-header sticky top-0 z-40 border-b">
    <div class="w-full max-w-[1480px] mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-4">
      <!-- Left: Logo & Docs Brand -->
      <div class="flex items-center gap-3">
        <button
          type="button"
          onclick={() => (mobileSidebarOpen = !mobileSidebarOpen)}
          class="docs-menu inline-flex size-9 items-center justify-center rounded-md border lg:hidden cursor-pointer"
          aria-label="Toggle navigation menu"
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"></line>
            <line x1="3" y1="6" x2="21" y2="6"></line>
            <line x1="3" y1="18" x2="21" y2="18"></line>
          </svg>
        </button>

        <button type="button" onclick={onback} class="docs-brand-button transition-opacity hover:opacity-75" aria-label="Back to starter home">
          <BrandLogo size="sm" class="docs-brand" />
        </button>

        <span class="docs-badge inline-flex items-center rounded-md border px-2 py-1 text-xs font-medium">Docs</span>
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
            aria-label="Search documentation"
            class="docs-search w-full h-9 pl-9 pr-14 rounded-md border text-sm focus:outline-none transition-colors"
          />
          <kbd class="docs-kbd absolute right-2.5 top-2 px-1.5 py-0.5 rounded border text-[10px] font-mono">Ctrl K</kbd>
        </div>
      </div>

      <!-- Right: Action Links -->
      <div class="flex items-center gap-2 text-xs font-medium">
        <button
          type="button"
          onclick={toggleTheme}
          class="docs-icon-button inline-flex size-9 items-center justify-center rounded-md border transition-colors"
          aria-label={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
          title={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
        >
          {#if theme === 'dark'}
            <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.93 4.93l1.42 1.42m11.3 11.3 1.42 1.42M2 12h2m16 0h2M4.93 19.07l1.42-1.42m11.3-11.3 1.42-1.42"/></svg>
          {:else}
            <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20.9 13A9 9 0 0 1 11 3.1 9 9 0 1 0 20.9 13Z"/></svg>
          {/if}
        </button>
        <!-- Exit to Starter Button -->
        <button
          type="button"
          onclick={onback}
          class="docs-exit h-9 px-3.5 rounded-md font-semibold transition-colors flex items-center gap-2 cursor-pointer"
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
        aria-label="Search documentation"
        class="docs-search w-full h-10 pl-10 pr-4 rounded-md border text-sm focus:outline-none"
      />
    </div>
  </div>

  <!-- Main Body: 3-Column Layout -->
  <div class="docs-layout w-full max-w-[1480px] mx-auto flex-1 flex">
    <!-- Left Navigation Sidebar -->
    <aside
      class="docs-sidebar fixed inset-y-0 left-0 z-50 w-72 p-6 overflow-y-auto transition-transform duration-200 ease-in-out lg:static lg:w-[270px] lg:p-6 {mobileSidebarOpen ? 'is-open' : ''}"
    >
      <!-- Mobile Close Button -->
      <div class="flex items-center justify-between pb-4 border-b lg:hidden mb-4">
        <span class="text-xs font-semibold uppercase tracking-wider docs-muted">Documentation</span>
        <button
          type="button"
          onclick={() => (mobileSidebarOpen = false)}
          class="docs-icon-button inline-flex size-8 items-center justify-center rounded-md transition-colors"
          aria-label="Close navigation"
        >
          <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="m18 6-12 12M6 6l12 12"/></svg>
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
                    class="docs-topic w-full text-left px-2.5 py-2 rounded-md text-[13px] transition-colors flex items-center justify-between cursor-pointer {activeTopicId === topic.id ? 'is-active font-semibold' : 'docs-muted'}"
                  >
                    <span class="truncate">{topic.title}</span>
                    {#if topic.badge}
                      <span class="docs-badge ml-2 text-[9px] rounded border px-1.5 py-0.5 font-medium">
                        {topic.badge}
                      </span>
                    {/if}
                  </button>
                </li>
              {/each}
            </ul>
          </div>
        {/each}
        {#if filteredCategories.length === 0}
          <p class="px-2 text-sm docs-muted">No topics match “{searchQuery}”. Try another term.</p>
        {/if}
      </nav>
    </aside>

    <!-- Backdrop for mobile sidebar -->
    {#if mobileSidebarOpen}
      <button
        type="button"
        onclick={() => (mobileSidebarOpen = false)}
                class="fixed inset-0 z-40 bg-black/60 lg:hidden"
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
            <span class="docs-badge rounded-md text-xs px-2 py-1 border">
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
            <h2 class="text-xl sm:text-2xl font-semibold docs-ink tracking-tight">
              {section.title}
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
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m12 3 1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8L12 3Z"/><path d="m19 15 .9 2.1L22 18l-2.1.9L19 21l-.9-2.1L16 18l2.1-.9L19 15Z"/></svg>
                  {:else if section.callout.type === 'warning'}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m10.3 3.9-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.7-3.1l-8-14a2 2 0 0 0-3.4 0Z"/><path d="M12 9v4m0 4h.01"/></svg>
                  {:else}
                    <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"/><path d="M12 16v-4m0-4h.01"/></svg>
                  {/if}
                </div>
                <div>{section.callout.text}</div>
              </div>
            {/if}

            <!-- Code Blocks with Filename Tab and Copy Button -->
            {#if section.code}
              {@const codeId = `${currentTopic.id}-${sIdx}`}
              <div class="docs-code rounded-lg border overflow-hidden">
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
              <div class="docs-table rounded-lg border overflow-x-auto text-sm">
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
    --background: #09090b;
    --foreground: #fafafa;
    --muted-surface: #18181b;
    --muted-foreground: #a1a1aa;
    --border: #27272a;
    --accent: #38bdf8;
    --accent-soft: #082f49;
    --code-background: #111113;
    --callout: #111113;
    background: var(--background);
    color: var(--foreground);
    color-scheme: dark;
    transition: background-color 180ms ease, color 180ms ease;
  }
  .docs-shell[data-theme='light'] {
    --background: #ffffff;
    --foreground: #09090b;
    --muted-surface: #f4f4f5;
    --muted-foreground: #71717a;
    --border: #e4e4e7;
    --accent: #0284c7;
    --accent-soft: #e0f2fe;
    --code-background: #18181b;
    --callout: #fafafa;
    color-scheme: light;
  }
  .docs-shell :global(button), .docs-shell :global(a), .docs-shell :global(input) { -webkit-tap-highlight-color: transparent; }
  .docs-header, .docs-sidebar, .docs-mobile-search { background: var(--background); border-color: var(--border); }
  .docs-sidebar { transform: translateX(-100%); }
  .docs-sidebar.is-open { transform: translateX(0); }
  .docs-brand-button { border: 0; padding: 0; background: transparent; cursor: pointer; }
  .docs-ink { color: var(--foreground); }
  :global(.docs-brand) { color: var(--foreground); }
  :global(.docs-brand > div) { background: var(--foreground); color: var(--background); box-shadow: none; }
  :global(.docs-brand > span) { color: var(--foreground); }
  .docs-muted { color: var(--muted-foreground); }
  .docs-body { color: var(--foreground); opacity: .82; }
  .docs-accent { color: var(--accent); }
  .docs-badge { color: var(--muted-foreground); border-color: var(--border); background: var(--muted-surface); }
  .docs-menu, .docs-icon-button { color: var(--foreground); border-color: var(--border); background: transparent; }
  .docs-menu:hover, .docs-icon-button:hover { background: var(--muted-surface); }
  .docs-search { color: var(--foreground); border-color: var(--border); background: var(--background); }
  .docs-search::placeholder { color: var(--muted-foreground); }
  .docs-search:focus { border-color: var(--accent); box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent) 22%, transparent); }
  .docs-kbd { color: var(--muted-foreground); border-color: var(--border); background: var(--muted-surface); }
  .docs-exit { color: var(--background); background: var(--foreground); }
  .docs-exit:hover { opacity: .85; }
  .docs-topic { color: var(--muted-foreground); }
  .docs-topic:hover { color: var(--foreground); background: var(--muted-surface); }
  .docs-topic.is-active { color: var(--foreground); background: var(--muted-surface); }
  .docs-topic.is-active::before { content: ''; width: 2px; height: 16px; border-radius: 1px; background: var(--accent); margin-right: 8px; flex: 0 0 auto; }
  .docs-topic.is-active > span:first-child { margin-right: auto; }
  .docs-topic:focus-visible, .docs-menu:focus-visible, .docs-icon-button:focus-visible, .docs-exit:focus-visible, .docs-pagination:focus-visible, .docs-copy:focus-visible, .docs-toc-link:focus-visible, .docs-brand-button:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
  .docs-section > h2 { scroll-margin-top: 6rem; }
  .docs-callout { color: var(--foreground); background: var(--callout); border-color: var(--border); border-radius: 8px; }
  .docs-callout.is-tip { background: color-mix(in srgb, var(--accent) 10%, var(--background)); border-color: color-mix(in srgb, var(--accent) 28%, var(--border)); }
  .docs-callout.is-warning { background: color-mix(in srgb, #f59e0b 9%, var(--background)); border-color: color-mix(in srgb, #f59e0b 25%, var(--border)); }
  .docs-callout.is-note { background: var(--muted-surface); }
  .docs-code { background: var(--code-background); border-color: var(--border); }
  .docs-code-text { color: #f1eee7; }
  .docs-copy { color: #d4d4d8; }
  .docs-table { border-color: var(--border); }
  .docs-table-head { color: var(--muted-foreground); background: var(--muted-surface); }
  .docs-table :global(th), .docs-table :global(td) { border-color: var(--border); }
  .docs-table-row:hover { background: var(--muted-surface); }
  .docs-pagination { color: var(--foreground); background: transparent; border-color: var(--border); border-radius: 8px; }
  .docs-pagination:hover { background: var(--muted-surface); border-color: color-mix(in srgb, var(--foreground) 18%, var(--border)); }
  .docs-toc { border-color: var(--border); }
  .docs-shell ::selection { color: var(--background); background: var(--accent); }
  .docs-shell :global(*) { scrollbar-color: var(--border) transparent; }
  @media (min-width: 1024px) {
    .docs-sidebar { transform: none; }
  }
  @media (prefers-reduced-motion: reduce) {
    .docs-shell *, .docs-shell *::before, .docs-shell *::after { scroll-behavior: auto !important; transition-duration: .01ms !important; animation-duration: .01ms !important; }
  }
</style>

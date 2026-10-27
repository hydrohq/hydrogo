<script lang="ts">
  import { tick, onMount } from 'svelte';
  import { router, docsHref } from './router.svelte';
  import { ui } from './ui.svelte';
  import { findPage, firstPage, headingsOf, neighbours } from './docs';
  import { setDocsSEO } from './seo';
  import Sidebar from './components/Sidebar.svelte';
  import Toc from './components/Toc.svelte';
  import DocBlock from './components/DocBlock.svelte';
  import Footer from './components/Footer.svelte';

  const route = $derived(router.current.name === 'docs' ? router.current : { name: 'docs' as const, section: '', page: '', anchor: '' });
  const entry = $derived(route.section ? findPage(route.section, route.page) : firstPage);
  const headings = $derived(entry ? headingsOf(entry.page) : []);
  const around = $derived(entry ? neighbours(entry) : {});

  let activeId = $state('');
  let title: HTMLHeadingElement | undefined = $state();
  let drawerClose: HTMLButtonElement | undefined = $state();

  // New page or anchor: scroll, move focus, update the document head.
  $effect(() => {
    const e = entry;
    const anchor = route.anchor;
    if (!e) {
      setDocsSEO('Page not found');
      return;
    }
    setDocsSEO(e.page.title, e.page.description);
    ui.drawerOpen = false;
    tick().then(() => {
      const target = anchor ? document.getElementById(anchor) : null;
      if (target) {
        target.scrollIntoView({ behavior: 'instant' as ScrollBehavior, block: 'start' });
        activeId = anchor;
      } else {
        window.scrollTo({ top: 0, behavior: 'instant' as ScrollBehavior });
        activeId = headings[0]?.id ?? '';
        title?.focus({ preventScroll: true });
      }
      spy();
    });
  });

  // Scroll-spy: the last heading above the top quarter of the viewport is active.
  let frame = 0;
  function spy() {
    cancelAnimationFrame(frame);
    frame = requestAnimationFrame(() => {
      let current = '';
      for (const h of headings) {
        const el = document.getElementById(h.id);
        if (el && el.getBoundingClientRect().top <= 110) current = h.id;
      }
      if (!current) current = headings[0]?.id ?? '';
      const atBottom = window.innerHeight + window.scrollY >= document.documentElement.scrollHeight - 4;
      if (atBottom && headings.length) current = headings[headings.length - 1].id;
      if (current !== activeId) activeId = current;
    });
  }

  onMount(() => {
    window.addEventListener('scroll', spy, { passive: true });
    window.addEventListener('resize', spy);
    return () => {
      window.removeEventListener('scroll', spy);
      window.removeEventListener('resize', spy);
    };
  });

  // Drawer: Escape closes, body scroll locks, focus moves in.
  $effect(() => {
    if (!ui.drawerOpen) return;
    document.body.style.overflow = 'hidden';
    tick().then(() => drawerClose?.focus());
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') ui.drawerOpen = false;
    };
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = '';
      window.removeEventListener('keydown', onKey);
    };
  });

  const sectionId = $derived(entry?.section.id ?? '');
  const pageId = $derived(entry?.page.id ?? '');
</script>

<div class="mx-auto flex max-w-[90rem]">
  <aside id="docs-sidebar" class="thin-scroll sticky top-14 hidden h-[calc(100vh-3.5rem)] w-64 shrink-0 overflow-y-auto border-r border-line py-8 pr-3 pl-4 lg:block xl:w-72 xl:pl-6">
    <Sidebar section={sectionId} page={pageId} />
  </aside>

  {#if ui.drawerOpen}
    <div class="fixed inset-0 z-50 lg:hidden" role="presentation">
      <button type="button" class="absolute inset-0 cursor-default bg-black/50" aria-label="Close navigation menu" tabindex="-1" onclick={() => (ui.drawerOpen = false)}></button>
      <div class="thin-scroll absolute inset-y-0 left-0 w-80 max-w-[85vw] overflow-y-auto bg-bg px-4 pb-8 shadow-2xl" role="dialog" aria-modal="true" aria-label="Documentation navigation">
        <div class="sticky top-0 -mx-4 mb-4 flex items-center justify-between border-b border-line bg-bg px-4 py-3">
          <span class="font-semibold text-fg">Documentation</span>
          <button
            bind:this={drawerClose}
            type="button"
            class="inline-flex size-9 items-center justify-center rounded-md text-muted hover:bg-subtle hover:text-fg"
            aria-label="Close navigation menu"
            onclick={() => (ui.drawerOpen = false)}
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
          </button>
        </div>
        <Sidebar section={sectionId} page={pageId} onnavigate={() => (ui.drawerOpen = false)} />
      </div>
    </div>
  {/if}

  <div class="min-w-0 flex-1">
    <div class="px-4 py-8 sm:px-8 lg:px-10 xl:flex xl:gap-10">
      {#if entry}
        <article class="mx-auto min-w-0 max-w-3xl flex-1 xl:mx-0">
          <nav aria-label="Breadcrumb" class="mb-3 text-sm text-faint">
            <ol class="flex flex-wrap items-center gap-1.5">
              <li><a href="#/docs" class="no-underline hover:text-fg">Docs</a></li>
              <li aria-hidden="true">/</li>
              <li><a href={docsHref(entry.section.id, entry.section.pages[0].id)} class="no-underline hover:text-fg">{entry.section.title}</a></li>
            </ol>
          </nav>
          <h1 bind:this={title} tabindex="-1" class="text-3xl font-bold tracking-tight text-fg sm:text-4xl" style="outline: none">{entry.page.title}</h1>
          <p class="mt-3 mb-8 text-lg leading-8 text-muted">{entry.page.description}</p>

          {#each entry.page.blocks as block}
            <DocBlock {block} section={entry.section.id} page={entry.page.id} />
          {/each}

          <nav aria-label="Previous and next page" class="mt-16 grid gap-3 border-t border-line pt-6 sm:grid-cols-2">
            {#if around.prev}
              <a href={docsHref(around.prev.section.id, around.prev.page.id)} class="group rounded-lg border border-line p-4 no-underline transition-colors hover:border-accent">
                <span class="block text-xs text-faint">← Previous</span>
                <span class="mt-1 block font-medium text-fg group-hover:text-accent">{around.prev.page.title}</span>
              </a>
            {:else}<span></span>{/if}
            {#if around.next}
              <a href={docsHref(around.next.section.id, around.next.page.id)} class="group rounded-lg border border-line p-4 text-right no-underline transition-colors hover:border-accent sm:col-start-2">
                <span class="block text-xs text-faint">Next →</span>
                <span class="mt-1 block font-medium text-fg group-hover:text-accent">{around.next.page.title}</span>
              </a>
            {/if}
          </nav>

          <p class="mt-8 text-sm text-faint">
            Found a mistake? Docs live in <code class="rounded border border-line bg-subtle px-1 py-0.5 font-mono text-xs">examples/svelte-starter/src/lib/docs/</code>;
            <a class="text-accent" href="https://github.com/ardaozyildirim/hydrogo/issues" target="_blank" rel="noopener noreferrer">open an issue</a> or send a pull request.
          </p>
        </article>

        <aside class="thin-scroll sticky top-20 hidden max-h-[calc(100vh-6rem)] w-56 shrink-0 self-start overflow-y-auto xl:block">
          <Toc section={entry.section.id} page={entry.page.id} {headings} {activeId} />
        </aside>
      {:else}
        <article class="mx-auto max-w-xl py-20 text-center">
          <p class="text-sm font-semibold text-accent">404</p>
          <h1 bind:this={title} tabindex="-1" class="mt-2 text-3xl font-bold text-fg" style="outline: none">Page not found</h1>
          <p class="mt-3 text-muted">That documentation page does not exist. Use the sidebar or search (Ctrl/Cmd+K) to find what you need.</p>
          <a href="#/docs" class="mt-6 inline-flex rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-fg no-underline">Back to the docs</a>
        </article>
      {/if}
    </div>
    <Footer />
  </div>
</div>

<script lang="ts">
  interface Props {
    onback: () => void;
  }

  let { onback }: Props = $props();

  let activeTab = $state<'start' | 'api' | 'content-types' | 'admin' | 'deployment'>('start');
</script>

<div class="max-w-4xl mx-auto space-y-8 pb-16">
  <!-- Top Bar -->
  <div class="flex items-center justify-between border-b border-slate-800 pb-4">
    <button
      type="button"
      onclick={onback}
      class="inline-flex items-center gap-2 text-sm text-slate-400 hover:text-white transition-colors cursor-pointer"
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="19" y1="12" x2="5" y2="12"></line>
        <polyline points="12 19 5 12 12 5"></polyline>
      </svg>
      Back to Starter
    </button>

    <a
      href="https://ardalabs.dev"
      target="_blank"
      rel="noreferrer"
      class="text-xs text-sky-400 hover:text-sky-300 flex items-center gap-1"
    >
      Visit ardalabs.dev
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
        <polyline points="15 3 21 3 21 9"></polyline>
        <line x1="10" y1="14" x2="21" y2="3"></line>
      </svg>
    </a>
  </div>

  <!-- Header Title -->
  <div class="space-y-2">
    <div class="inline-flex items-center gap-2 px-2.5 py-1 rounded-full bg-sky-500/10 border border-sky-500/20 text-sky-400 text-xs font-mono">
      Documentation
    </div>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-white tracking-tight">HydroGo Guide & Reference</h1>
    <p class="text-sm text-slate-400 leading-relaxed">
      Learn how to develop, customize, and deploy your headless CMS with Go, Svelte 5, and REST APIs.
    </p>
  </div>

  <!-- Navigation Tabs -->
  <div class="flex flex-wrap gap-2 border-b border-slate-800 pb-3">
    <button
      type="button"
      onclick={() => (activeTab = 'start')}
      class="px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer {activeTab === 'start' ? 'bg-white text-black font-semibold' : 'text-slate-400 hover:text-white hover:bg-slate-900'}"
    >
      Quick Start
    </button>
    <button
      type="button"
      onclick={() => (activeTab = 'api')}
      class="px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer {activeTab === 'api' ? 'bg-white text-black font-semibold' : 'text-slate-400 hover:text-white hover:bg-slate-900'}"
    >
      REST API
    </button>
    <button
      type="button"
      onclick={() => (activeTab = 'content-types')}
      class="px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer {activeTab === 'content-types' ? 'bg-white text-black font-semibold' : 'text-slate-400 hover:text-white hover:bg-slate-900'}"
    >
      Content Types (YAML)
    </button>
    <button
      type="button"
      onclick={() => (activeTab = 'admin')}
      class="px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer {activeTab === 'admin' ? 'bg-white text-black font-semibold' : 'text-slate-400 hover:text-white hover:bg-slate-900'}"
    >
      Admin Studio
    </button>
    <button
      type="button"
      onclick={() => (activeTab = 'deployment')}
      class="px-3.5 py-1.5 rounded-lg text-xs font-medium transition-colors cursor-pointer {activeTab === 'deployment' ? 'bg-white text-black font-semibold' : 'text-slate-400 hover:text-white hover:bg-slate-900'}"
    >
      Deployment
    </button>
  </div>

  <!-- Tab Contents -->
  <div class="space-y-6">
    {#if activeTab === 'start'}
      <section class="space-y-4">
        <h2 class="text-xl font-bold text-white">Getting Started with HydroGo</h2>
        <p class="text-sm text-slate-300 leading-relaxed">
          HydroGo is a single-binary headless CMS. It manages your content, assets, and custom data types, exposing them through clean JSON endpoints for modern frontends.
        </p>

        <div class="space-y-3 pt-2">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-400">1. Run Local Development Server</h3>
          <div class="p-4 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs text-sky-300 overflow-x-auto">
            go run ./tools/devserver
          </div>
          <p class="text-xs text-slate-400">
            This starts the Go backend, Vite with HMR for the Svelte admin and frontend, and provides live reload on port <code class="text-white">8090</code>.
          </p>
        </div>

        <div class="space-y-3 pt-2">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-400">2. Log into Admin Studio</h3>
          <p class="text-xs text-slate-300">
            Open <a href="/admin/" class="text-sky-400 hover:underline font-mono">/admin</a> in your browser. Admin Studio lets you manage pages, news articles, media uploads, and custom content types.
          </p>
        </div>

        <div class="space-y-3 pt-2">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-400">3. Edit Frontend Views</h3>
          <p class="text-xs text-slate-300">
            Edit files in <code class="bg-white/10 text-white px-1.5 py-0.5 rounded font-mono text-xs">examples/svelte-starter/src/</code>. Any changes update instantly in your browser via Vite HMR.
          </p>
        </div>
      </section>

    {:else if activeTab === 'api'}
      <section class="space-y-4">
        <h2 class="text-xl font-bold text-white">Public REST API Reference</h2>
        <p class="text-sm text-slate-300 leading-relaxed">
          All endpoints return JSON wrapped in standard envelopes: <code class="text-sky-300 font-mono text-xs">&#123; "data": [...], "meta": &#123; ... &#125; &#125;</code>.
        </p>

        <div class="border border-slate-800 rounded-xl overflow-hidden divide-y divide-slate-800 text-xs">
          <div class="p-4 bg-slate-900/60 flex items-start justify-between gap-4">
            <div>
              <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold mr-2">GET</span>
              <code class="text-white font-mono">/api/v1/pages?locale=en</code>
              <p class="text-slate-400 mt-1">Fetches published pages for the requested locale.</p>
            </div>
          </div>

          <div class="p-4 bg-slate-900/60 flex items-start justify-between gap-4">
            <div>
              <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold mr-2">GET</span>
              <code class="text-white font-mono">/api/v1/pages/by-slug?locale=en&slug=about</code>
              <p class="text-slate-400 mt-1">Fetches a single page by slug with structured content blocks.</p>
            </div>
          </div>

          <div class="p-4 bg-slate-900/60 flex items-start justify-between gap-4">
            <div>
              <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold mr-2">GET</span>
              <code class="text-white font-mono">/api/v1/news?locale=en&limit=10</code>
              <p class="text-slate-400 mt-1">Fetches published news articles, sorted by publish date.</p>
            </div>
          </div>

          <div class="p-4 bg-slate-900/60 flex items-start justify-between gap-4">
            <div>
              <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold mr-2">GET</span>
              <code class="text-white font-mono">/api/v1/content/:type</code>
              <p class="text-slate-400 mt-1">Fetches custom content type items (e.g. <code class="text-sky-300">/api/v1/content/faq</code>).</p>
            </div>
          </div>

          <div class="p-4 bg-slate-900/60 flex items-start justify-between gap-4">
            <div>
              <span class="px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 font-mono font-bold mr-2">GET</span>
              <code class="text-white font-mono">/readyz</code>
              <p class="text-slate-400 mt-1">Health check & readiness probe for load balancers and orchestrators.</p>
            </div>
          </div>
        </div>
      </section>

    {:else if activeTab === 'content-types'}
      <section class="space-y-4">
        <h2 class="text-xl font-bold text-white">Declarative Content Types</h2>
        <p class="text-sm text-slate-300 leading-relaxed">
          Create new content collections (like FAQs, Team Members, Portfolios) by adding a single YAML file in <code class="text-sky-300 font-mono">content-types/</code>. No Go code or recompilation required!
        </p>

        <div class="p-4 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs text-slate-300 overflow-x-auto leading-relaxed">
          <pre><code>id: faq
name: FAQ
singular: Question
plural: Questions
public: true
list:
  search: [question, answer]
  default_sort: -created_at
fields:
  - id: question
    name: Question
    type: text
    required: true
    localized: true
  - id: answer
    name: Answer
    type: rich_text
    required: true
    localized: true
  - id: category
    name: Category
    type: select
    options: [General, Billing, Technical]</code></pre>
        </div>
        <p class="text-xs text-slate-400">
          HydroGo instantly creates admin UI at <code class="text-white">/admin/content/faq</code> and a public API endpoint at <code class="text-white">/api/v1/content/faq</code>.
        </p>
      </section>

    {:else if activeTab === 'admin'}
      <section class="space-y-4">
        <h2 class="text-xl font-bold text-white">Admin Studio</h2>
        <p class="text-sm text-slate-300 leading-relaxed">
          Admin Studio is a high-performance Single Page Application built on Svelte 5 and shadcn-svelte.
        </p>
        <ul class="list-disc list-inside space-y-2 text-xs text-slate-300">
          <li><strong>Zero Latency:</strong> Embedded directly inside the Go binary.</li>
          <li><strong>Media Management:</strong> Automatic image optimization, WebP compression, focal point adjustment, and BlurHash previews.</li>
          <li><strong>Role-Based Access:</strong> Admin, Editor, and Viewer permissions with granular overrides.</li>
          <li><strong>Revisions & Drafts:</strong> Edit draft pages safely before publishing to the live website.</li>
        </ul>
      </section>

    {:else if activeTab === 'deployment'}
      <section class="space-y-4">
        <h2 class="text-xl font-bold text-white">Production Deployment</h2>
        <p class="text-sm text-slate-300 leading-relaxed">
          HydroGo compiles into a single, self-contained binary.
        </p>

        <div class="space-y-2">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-400">Build Single Binary</h3>
          <div class="p-4 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs text-sky-300">
            go build -o bin/hydro ./cmd/hydro
          </div>
        </div>

        <div class="space-y-2">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-400">Run Database Migrations</h3>
          <div class="p-4 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs text-sky-300">
            ./bin/hydro migrate --config config.yaml
          </div>
        </div>

        <div class="space-y-2">
          <h3 class="text-xs uppercase font-mono tracking-wider text-slate-400">Start Server</h3>
          <div class="p-4 rounded-xl bg-slate-900 border border-slate-800 font-mono text-xs text-sky-300">
            ./bin/hydro serve --config config.yaml
          </div>
        </div>
      </section>
    {/if}
  </div>
</div>

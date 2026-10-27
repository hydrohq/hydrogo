<script lang="ts">
  import Logo from './components/Logo.svelte';
  import CodeBlock from './components/CodeBlock.svelte';
  import Diagram from './components/Diagram.svelte';
  import Footer from './components/Footer.svelte';
  import { sections } from './docs';
  import { docsHref } from './router.svelte';

  const quick = `curl -fsSL https://github.com/ardaozyildirim/hydrogo/releases/latest/download/install.sh | sh
hydro new mysite --preset blog --admin-email you@example.com
cd mysite && hydro serve --config config.yaml`;

  const features = [
    { title: 'One binary', text: 'API, admin panel, migrations and workers ship as a single executable. Copy it, run it, done.', href: docsHref('introduction', 'architecture') },
    { title: 'Zero build on publish', text: 'Publishing is live on the next API request. No rebuild, no deploy, no static regeneration.', href: docsHref('introduction', 'architecture', 'zero-build-on-publish') },
    { title: 'YAML content types', text: 'Add FAQs, team pages or events with one YAML file. Admin screens and a public API come with it.', href: docsHref('content-types', 'overview') },
    { title: 'REST API', text: 'Pages, news, media and content types over JSON with one envelope, one error shape and request ids.', href: docsHref('rest-api', 'overview') },
    { title: 'Embedded admin', text: 'A Svelte admin panel with roles, revisions, a media library, audit trail and an English and Turkish UI.', href: docsHref('admin-guide', 'overview') },
    { title: 'Multi-language', text: 'Per-locale content fields, a ?locale= parameter and fallback to the default locale.', href: docsHref('multi-language', 'content-locales') },
    { title: 'Any database', text: 'SQLite by default; PostgreSQL and MySQL/MariaDB when you grow. Optional Redis and S3.', href: docsHref('deployment', 'databases-and-storage') },
    { title: 'Secure by default', text: 'CSRF protection, MFA, scoped API keys, rate limits, strict headers and a searchable audit trail.', href: docsHref('security', 'hardening') },
    { title: 'Typed client', text: '@hydrogo/client has no dependencies and runs in browsers, Node, Deno, Bun and edge runtimes.', href: docsHref('frontends', 'client') },
  ];
</script>

<div>
  <section class="relative overflow-hidden border-b border-line">
    <div class="pointer-events-none absolute inset-0 -z-0 opacity-60 [background:radial-gradient(60rem_28rem_at_50%_-10%,var(--accent-soft),transparent)]" aria-hidden="true"></div>
    <div class="relative mx-auto max-w-4xl px-4 pt-16 pb-14 text-center sm:px-6 sm:pt-24 sm:pb-20">
      <div class="mb-6 flex justify-center"><Logo size="lg" /></div>
      <p class="mx-auto mb-5 inline-flex rounded-full border border-line bg-subtle px-3 py-1 text-xs font-medium text-muted">Single binary · Headless · MIT licensed</p>
      <h1 class="text-4xl font-bold tracking-tight text-fg sm:text-6xl">A headless CMS in one Go binary</h1>
      <p class="mx-auto mt-5 max-w-2xl text-lg leading-8 text-muted">
        HydroGo manages pages, news, media, members and your own content types, and serves them over a clean REST API. It does not render a website: you build the frontend you want, and publishing never needs a rebuild.
      </p>
      <div class="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <a href={docsHref('getting-started', 'quick-start')} class="inline-flex h-11 w-full items-center justify-center rounded-lg bg-accent px-6 text-sm font-semibold text-accent-fg no-underline hover:opacity-90 sm:w-auto">Get started</a>
        <a href="#/docs" class="inline-flex h-11 w-full items-center justify-center rounded-lg border border-line-strong bg-bg px-6 text-sm font-semibold text-fg no-underline hover:bg-subtle sm:w-auto">Read the docs</a>
        <a href="/admin/" class="inline-flex h-11 w-full items-center justify-center rounded-lg px-6 text-sm font-medium text-muted no-underline hover:text-fg sm:w-auto">Open the admin →</a>
      </div>
    </div>
  </section>

  <section class="mx-auto max-w-3xl px-4 py-14 sm:px-6" aria-labelledby="quick-start-title">
    <h2 id="quick-start-title" class="text-center text-2xl font-semibold tracking-tight text-fg">Running in three commands</h2>
    <p class="mt-2 text-center text-muted">Install, scaffold a project with an admin user, and serve it.</p>
    <div class="text-left"><CodeBlock code={quick} lang="bash" title="terminal" /></div>
    <p class="text-center text-sm text-muted">
      Then open <code class="rounded border border-line bg-subtle px-1.5 py-0.5 font-mono text-xs">http://127.0.0.1:8080/admin</code> for the admin panel and
      <code class="rounded border border-line bg-subtle px-1.5 py-0.5 font-mono text-xs">/api/v1</code> for the API. Other platforms: see <a class="font-medium text-accent" href={docsHref('getting-started', 'install')}>Install</a>.
    </p>
  </section>

  <section class="mx-auto max-w-6xl px-4 pb-14 sm:px-6" aria-labelledby="features-title">
    <h2 id="features-title" class="text-center text-2xl font-semibold tracking-tight text-fg">Everything a content backend needs</h2>
    <ul class="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {#each features as f}
        <li>
          <a href={f.href} class="flex h-full flex-col rounded-xl border border-line bg-elevated p-5 no-underline transition-colors hover:border-accent">
            <span class="font-semibold text-fg">{f.title}</span>
            <span class="mt-1.5 text-sm leading-6 text-muted">{f.text}</span>
          </a>
        </li>
      {/each}
    </ul>
  </section>

  <section class="border-y border-line bg-subtle" aria-labelledby="arch-title">
    <div class="mx-auto max-w-4xl px-4 py-14 sm:px-6">
      <h2 id="arch-title" class="text-center text-2xl font-semibold tracking-tight text-fg">How it fits together</h2>
      <p class="mx-auto mt-2 max-w-2xl text-center text-muted">Editors work in the embedded admin; your frontends read the API. Go serves JSON, never HTML.</p>
      <Diagram name="architecture" />
    </div>
  </section>

  <section class="mx-auto max-w-6xl px-4 py-14 sm:px-6" aria-labelledby="explore-title">
    <h2 id="explore-title" class="text-center text-2xl font-semibold tracking-tight text-fg">Explore the documentation</h2>
    <div class="mt-8 grid gap-x-8 gap-y-6 sm:grid-cols-2 lg:grid-cols-4">
      {#each sections as s}
        <div>
          <h3 class="mb-2 text-sm font-semibold text-fg"><a class="no-underline hover:text-accent" href={docsHref(s.id, s.pages[0].id)}>{s.title}</a></h3>
          <ul class="space-y-1 text-sm">
            {#each s.pages.slice(0, 4) as p}
              <li><a class="text-muted no-underline hover:text-accent" href={docsHref(s.id, p.id)}>{p.title}</a></li>
            {/each}
            {#if s.pages.length > 4}<li class="text-faint">and {s.pages.length - 4} more…</li>{/if}
          </ul>
        </div>
      {/each}
    </div>
  </section>

  <Footer />
</div>

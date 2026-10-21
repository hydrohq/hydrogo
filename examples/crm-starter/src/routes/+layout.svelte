<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import { messages } from '$lib/i18n';
  let { data, children } = $props();
  const t = $derived(messages[data.lang]);
  const links = $derived([
    ['/', t.dashboard],
    ['/deals', t.deals],
    ['/companies', t.companies],
    ['/contacts', t.contacts],
    ['/tasks', t.tasks]
  ] as const);
  const current = (href: string) => (href === '/' ? page.url.pathname === '/' : page.url.pathname.startsWith(href));
</script>

{#if data.email}
  <header class="top">
    <strong>{t.appName}</strong>
    <nav>
      {#each links as [href, label]}<a {href} aria-current={current(href) ? 'page' : undefined}>{label}</a>{/each}
    </nav>
    <div class="who">
      <form method="POST" action="/lang" class="inline" style="margin:0">
        <input type="hidden" name="back" value={page.url.pathname + page.url.search} />
        <select name="lang" aria-label="Language" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
          <option value="en" selected={data.lang === 'en'}>EN</option>
          <option value="tr" selected={data.lang === 'tr'}>TR</option>
        </select>
        <noscript><button>OK</button></noscript>
      </form>
      <span>{data.email}</span>
      <form method="POST" action="/logout" style="margin:0"><button>{t.signOut}</button></form>
    </div>
  </header>
{/if}

{@render children()}

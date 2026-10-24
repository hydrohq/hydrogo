<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import { messages } from '$lib/i18n';
  import Avatar from '$lib/components/Avatar.svelte';
  let { data, children } = $props();
  const t = $derived(messages[data.lang]);

  const links = $derived([
    { href: '/', label: t.dashboard, icon: 'home' },
    { href: '/deals', label: t.deals, icon: 'deal' },
    { href: '/companies', label: t.companies, icon: 'building' },
    { href: '/contacts', label: t.contacts, icon: 'users' },
    { href: '/tasks', label: t.tasks, icon: 'tasks' }
  ] as const);
  const current = (href: string) => (href === '/' ? page.url.pathname === '/' : page.url.pathname.startsWith(href));

  const icons: Record<string, string> = {
    home: 'M3 10.5 12 3l9 7.5M5 9.5V20a1 1 0 0 0 1 1h4v-6h4v6h4a1 1 0 0 0 1-1V9.5',
    deal: 'M12 2 3 7v10l9 5 9-5V7l-9-5ZM3 7l9 5 9-5M12 12v10',
    building: 'M4 21V5a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v16M14 21v-9a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v9M4 21h16M7 7h1M7 11h1M7 15h1',
    users: 'M7 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6ZM2.5 20c.5-3.2 2.7-5 4.5-5s4 1.8 4.5 5M16.5 11a2.5 2.5 0 1 0 0-5M15 20c.3-2.7 2-4.3 3.5-4.7a4 4 0 0 1 3 .9',
    tasks: 'm4 12 4 4L20 4M4 20h10',
    logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9'
  };
</script>

<div class="shell">
  {#if data.email}
    <aside class="sidebar">
      <div class="brand">
        <span class="mark">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="9" cy="9" r="4" /><circle cx="16" cy="13" r="4" />
          </svg>
        </span>
      </div>
      <nav class="sidenav">
        {#each links as l}
          <a href={l.href} aria-current={current(l.href) ? 'page' : undefined}>
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
              <path d={icons[l.icon]} />
            </svg>
            <span class="tooltip">{l.label}</span>
          </a>
        {/each}
      </nav>
      <div class="who">
        <div title={data.email}><Avatar name={data.email} /></div>
        <form method="POST" action="/lang" style="margin:0">
          <input type="hidden" name="back" value={page.url.pathname + page.url.search} />
          <select name="lang" aria-label="Language" onchange={(e) => e.currentTarget.form?.requestSubmit()}>
            <option value="en" selected={data.lang === 'en'}>EN</option>
            <option value="tr" selected={data.lang === 'tr'}>TR</option>
          </select>
        </form>
        <form method="POST" action="/logout">
          <button aria-label={t.signOut} title={t.signOut}>
            <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d={icons.logout} /></svg>
          </button>
        </form>
      </div>
    </aside>
  {/if}

  {@render children()}
</div>

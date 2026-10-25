<script lang="ts">
  import '../app.css';
  import { page } from '$app/state';
  import { fly } from 'svelte/transition';
  import { ease, ms } from '$lib/motion';
  import { messages } from '$lib/i18n';
  import { firstName, greetingKey } from '$lib/greeting';
  import Avatar from '$lib/components/Avatar.svelte';
  import Icon from '$lib/components/Icon.svelte';
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
  const section = $derived(links.find((l) => current(l.href)));

  // The greeting follows the visitor's clock, so it is set after mount.
  let hour = $state(new Date().getHours());
  $effect(() => { hour = new Date().getHours(); });
  const isHome = $derived(page.url.pathname === '/');
  const title = $derived(isHome ? `${t[greetingKey(hour)]}, ${firstName(data.email)}!` : (section?.label ?? t.appName));

  let menu = $state(false);
  $effect(() => { void page.url.pathname; menu = false; });
</script>

<svelte:window onclick={(e) => { if (!(e.target as Element).closest('.account')) menu = false; }} onkeydown={(e) => e.key === 'Escape' && (menu = false)} />

{#if data.email}
  <div class="shell">
    <aside class="rail">
      <a class="mark" href="/" aria-label={t.appName}>
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="9" r="4" /><circle cx="16" cy="13" r="4" /></svg>
      </a>
      <nav class="sidenav" aria-label={t.appName}>
        {#each links as l (l.href)}
          <a href={l.href} aria-current={current(l.href) ? 'page' : undefined} aria-label={l.label}>
            <Icon name={l.icon} />
            <span class="tooltip">{l.label}</span>
          </a>
        {/each}
      </nav>
      <form method="POST" action="/logout">
        <button class="railbtn" aria-label={t.signOut}><Icon name="logout" /><span class="tooltip">{t.signOut}</span></button>
      </form>
    </aside>

    <div class="main">
      <header class="top">
        <div class="hello">
          <h1>{title}</h1>
          {#if isHome}<p class="muted">{t.dashboardSub}</p>{/if}
        </div>
        <div class="account">
          <button aria-haspopup="menu" aria-expanded={menu} aria-label={t.account} onclick={() => (menu = !menu)}><Avatar name={data.email} size="nav" /></button>
          {#if menu}
            <div class="menu" role="menu" transition:fly={{ y: -8, duration: ms(170), easing: ease }}>
              <strong>{data.email}</strong>
              <form method="POST" action="/lang" class="langs" aria-label={t.language}>
                <input type="hidden" name="back" value={page.url.pathname + page.url.search} />
                <button name="lang" value="en" aria-pressed={data.lang === 'en'}>English</button>
                <button name="lang" value="tr" aria-pressed={data.lang === 'tr'}>Türkçe</button>
              </form>
              <form method="POST" action="/logout"><button class="out" style="width:100%"><Icon name="logout" size={16} /> {t.signOut}</button></form>
            </div>
          {/if}
        </div>
      </header>

      <main class="content">
        {#key page.url.pathname}
          <div in:fly={{ y: 12, duration: ms(280), easing: ease }}>{@render children()}</div>
        {/key}
      </main>
    </div>

    <nav class="tabbar" aria-label={t.appName}>
      {#each links as l (l.href)}
        <a href={l.href} aria-current={current(l.href) ? 'page' : undefined} aria-label={l.label}><Icon name={l.icon} /></a>
      {/each}
    </nav>
  </div>
{:else}
  <div class="shell bare">{@render children()}</div>
{/if}

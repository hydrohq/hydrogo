<script lang="ts">
  import { page } from '$app/state';
  import { localePrefix, switchLocalePath } from '$lib/config';

  let {
    siteName,
    navigation,
    locale,
    locales
  }: { siteName: string; navigation: { label: string; href: string }[]; locale: string; locales: string[] } = $props();

  const prefix = $derived(localePrefix(locale));
  // Internal links get the language prefix; absolute links are left alone.
  const link = (href: string) => (href.startsWith('/') ? `${prefix}${href === '/' ? '' : href}` || '/' : href);
  const links = $derived(
    navigation.length > 0
      ? navigation
      : [
          { label: 'Home', href: '/' },
          { label: 'News', href: '/news' }
        ]
  );
</script>

<header class="site-header">
  <a class="brand" href={prefix || '/'}>{siteName}</a>
  <nav aria-label="Main">
    {#each links as item (item.href)}
      <a href={link(item.href)} aria-current={page.url.pathname === link(item.href) ? 'page' : undefined}>{item.label}</a>
    {/each}
  </nav>
  {#if locales.length > 1}
    <nav class="languages" aria-label="Language">
      {#each locales as code (code)}
        <a href={switchLocalePath(page.url.pathname, code)} hreflang={code} aria-current={code === locale ? 'true' : undefined}>{code.toUpperCase()}</a>
      {/each}
    </nav>
  {/if}
</header>

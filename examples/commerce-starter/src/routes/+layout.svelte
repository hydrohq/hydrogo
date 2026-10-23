<script lang="ts">
  import '../app.css';
  import type { Snippet } from 'svelte';
  import { page } from '$app/state';
  import type { LayoutData } from './$types';

  let { data, children }: { data: LayoutData; children: Snippet } = $props();

  const allCurrencies = $derived([data.currencies.default, ...data.currencies.display]);
</script>

<a href="#main" class="skip">İçeriğe geç</a>

<header class="shell-header">
  <a class="brand" href="/">Deri Atölyesi</a>
  <nav>
    {#each data.categories as category (category.id)}
      <a href={`/kategori/${category.slug}`}>{category.name}</a>
    {/each}
  </nav>
  <form class="search-form" action="/arama" method="GET">
    <input type="search" name="q" placeholder="Ürün ara…" aria-label="Ürün ara" />
  </form>
  {#if allCurrencies.length > 1}
    <div class="currency-switch">
      {#each allCurrencies as code (code)}
        <a href={`/para-birimi/${code}?redirect=${encodeURIComponent(page.url.pathname + page.url.search)}`} aria-current={data.displayCurrency === code ? 'true' : undefined}>
          {code}
        </a>
      {/each}
    </div>
  {/if}
  <a class="cart-link" href="/sepet">
    Sepet
    {#if data.cartItemCount > 0}<span class="count">{data.cartItemCount}</span>{/if}
  </a>
</header>

<main id="main">
  {@render children()}
</main>

<footer class="shell-footer">
  <p>Bu, HydroGo'nun commerce-starter örneğidir -- gerçek bir mağaza değildir.</p>
</footer>

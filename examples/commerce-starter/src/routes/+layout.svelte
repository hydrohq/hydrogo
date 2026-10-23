<script lang="ts">
  import '../app.css';
  import type { Snippet } from 'svelte';
  import type { LayoutData } from './$types';

  let { data, children }: { data: LayoutData; children: Snippet } = $props();
</script>

<a href="#main" class="skip">İçeriğe geç</a>

<header class="shell-header">
  <a class="brand" href="/">Deri Atölyesi</a>
  <nav>
    {#each data.categories as category (category.id)}
      <a href={`/?kategori=${encodeURIComponent(category.slug)}`}>{category.name}</a>
    {/each}
  </nav>
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

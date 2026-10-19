<script lang="ts">
  let { children } = $props();
</script>

<header>
  <a href="/">Home</a>
  <a href="/news">News</a>
</header>
<main>{@render children()}</main>

<style>
  :global(body) { font: 16px/1.6 system-ui, sans-serif; margin: 0; color: #1a1a1a; }
  header { display: flex; gap: 1rem; padding: 1rem 1.5rem; border-bottom: 1px solid #e5e5e5; }
  main { max-width: 46rem; margin: 0 auto; padding: 1.5rem; }
  a { color: #0b57d0; }
</style>

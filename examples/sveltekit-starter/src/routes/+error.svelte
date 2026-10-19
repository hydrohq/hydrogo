<script lang="ts">
  import { page } from '$app/state';
</script>

<svelte:head><title>{page.status} · {page.error?.message}</title><meta name="robots" content="noindex" /></svelte:head>

<main class="error">
  <h1>{page.status}</h1>
  <p>{page.status === 404 ? 'This page could not be found.' : page.error?.message}</p>
  <p><a href="/">Back to the home page</a></p>
</main>

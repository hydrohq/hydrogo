<script lang="ts">
  import { page } from '$app/state';
</script>

<svelte:head><title>{page.status} · Hata</title><meta name="robots" content="noindex" /></svelte:head>

<main class="error" style="text-align:center; padding-top:4rem;">
  <h1 style="font-size:4rem; margin:0;">{page.status}</h1>
  <p>{page.status === 404 ? 'Bu sayfa bulunamadı.' : page.error?.message}</p>
  <p><a href="/">Ana sayfaya dön</a></p>
</main>

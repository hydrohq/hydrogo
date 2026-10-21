<script lang="ts">
  import { page } from '$app/state';
</script>

<main class="page">
  <h1>{page.status}</h1>
  <p class="error">{page.error?.message}</p>
  <p><a href="/">←</a></p>
</main>

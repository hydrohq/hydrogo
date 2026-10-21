<script lang="ts">
  import { enhance } from '$app/forms';
  import FormError from '$lib/components/FormError.svelte';
  import { messages } from '$lib/i18n';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.companies} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.companies}</h1>
  <form method="GET" class="inline"><label>{t.search}<input name="q" value={data.search} type="search" /></label><button>{t.search}</button></form>
  <form method="POST" action="?/create" class="inline" use:enhance>
    <label>{t.name}<input name="name" required /></label>
    <label>{t.industry}<select name="industry"><option value=""></option>{#each ['software', 'manufacturing', 'retail', 'services', 'other'] as value}<option {value}>{value}</option>{/each}</select></label>
    <label>{t.website}<input name="website" /></label>
    <button class="primary">{t.newCompany}</button>
  </form>
  <FormError {form} lang={data.lang} />
  <div class="table-wrap">
    <table>
      <thead><tr><th>{t.name}</th><th>{t.industry}</th><th>{t.owner}</th></tr></thead>
      <tbody>
        {#each data.companies as company}
          <tr><td><a href={`/companies/${company.id}`}>{company.name}</a></td><td>{company.industry ?? ''}</td><td>{company.owner ?? ''}</td></tr>
        {:else}
          <tr><td colspan="3" class="muted">{t.empty}</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
  {#if data.next}<p><a href={`?after=${encodeURIComponent(data.next)}${data.search ? `&q=${encodeURIComponent(data.search)}` : ''}`}>{t.loadMore}</a></p>{/if}
</main>

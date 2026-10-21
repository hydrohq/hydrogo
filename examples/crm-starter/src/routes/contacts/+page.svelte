<script lang="ts">
  import { enhance } from '$app/forms';
  import FormError from '$lib/components/FormError.svelte';
  import { messages } from '$lib/i18n';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.contacts} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.contacts}</h1>
  <form method="GET" class="inline"><label>{t.search}<input name="q" value={data.search} type="search" /></label><button>{t.search}</button></form>
  <form method="POST" action="?/create" class="inline" use:enhance>
    <label>{t.name}<input name="full_name" required /></label>
    <label>{t.email}<input name="email" type="email" /></label>
    <label>{t.company}<select name="company"><option value=""></option>{#each data.companies as company}<option value={company.id}>{company.name}</option>{/each}</select></label>
    <button class="primary">{t.newContact}</button>
  </form>
  <FormError {form} lang={data.lang} />
  <div class="table-wrap">
    <table>
      <thead><tr><th>{t.name}</th><th>{t.email}</th><th>{t.company}</th></tr></thead>
      <tbody>
        {#each data.contacts as contact}
          {@const company = contact.expanded?.company as { id: string; name?: string } | undefined}
          <tr><td>{contact.full_name}</td><td>{contact.email ?? ''}</td><td>{#if company}<a href={`/companies/${company.id}`}>{company.name}</a>{/if}</td></tr>
        {:else}
          <tr><td colspan="3" class="muted">{t.empty}</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
  {#if data.next}<p><a href={`?after=${encodeURIComponent(data.next)}${data.search ? `&q=${encodeURIComponent(data.search)}` : ''}`}>{t.loadMore}</a></p>{/if}
</main>

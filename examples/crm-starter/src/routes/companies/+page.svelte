<script lang="ts">
  import { enhance } from '$app/forms';
  import FormError from '$lib/components/FormError.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import { messages } from '$lib/i18n';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.companies} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.companies}</h1>
  <form method="GET" class="inline"><label>{t.search}<input name="q" value={data.search} type="search" /></label><button>{t.search}</button></form>
  <div class="panel-form">
    <form method="POST" action="?/create" class="inline" use:enhance style="margin:0">
      <label>{t.name}<input name="name" required /></label>
      <label>{t.industry}<select name="industry"><option value=""></option>{#each ['software', 'manufacturing', 'retail', 'services', 'other'] as value}<option {value}>{value}</option>{/each}</select></label>
      <label>{t.website}<input name="website" /></label>
      <button class="primary">{t.newCompany}</button>
    </form>
  </div>
  <FormError {form} lang={data.lang} />
  <div class="section-card">
    <div class="rowlist">
      {#each data.companies as company}
        <div class="rowcard">
          <Avatar name={company.name as string} />
          <div class="main">
            <a class="title" href={`/companies/${company.id}`}>{company.name}</a>
            <div class="sub">{[company.industry, company.owner].filter(Boolean).join(' · ')}</div>
          </div>
        </div>
      {:else}
        <div class="empty-state">{t.empty}</div>
      {/each}
    </div>
  </div>
  {#if data.next}<p><a href={`?after=${encodeURIComponent(data.next)}${data.search ? `&q=${encodeURIComponent(data.search)}` : ''}`}>{t.loadMore}</a></p>{/if}
</main>

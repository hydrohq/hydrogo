<script lang="ts">
  import { enhance } from '$app/forms';
  import FormError from '$lib/components/FormError.svelte';
  import Avatar from '$lib/components/Avatar.svelte';
  import { messages } from '$lib/i18n';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);
</script>

<svelte:head><title>{t.contacts} · {t.appName}</title></svelte:head>

<main class="page">
  <h1>{t.contacts}</h1>
  <form method="GET" class="inline"><label>{t.search}<input name="q" value={data.search} type="search" /></label><button>{t.search}</button></form>
  <div class="panel-form">
    <form method="POST" action="?/create" class="inline" use:enhance style="margin:0">
      <label>{t.name}<input name="full_name" required /></label>
      <label>{t.email}<input name="email" type="email" /></label>
      <label>{t.company}<select name="company"><option value=""></option>{#each data.companies as company}<option value={company.id}>{company.name}</option>{/each}</select></label>
      <button class="primary">{t.newContact}</button>
    </form>
  </div>
  <FormError {form} lang={data.lang} />
  <div class="section-card">
    <div class="rowlist">
      {#each data.contacts as contact}
        {@const company = contact.expanded?.company as { id: string; name?: string } | undefined}
        <div class="rowcard">
          <Avatar name={contact.full_name as string} />
          <div class="main">
            <span class="title">{contact.full_name}</span>
            <div class="sub">{[contact.email, company?.name].filter(Boolean).join(' · ')}</div>
          </div>
          {#if company}<a class="trailing" href={`/companies/${company.id}`}>{company.name}</a>{/if}
        </div>
      {:else}
        <div class="empty-state">{t.empty}</div>
      {/each}
    </div>
  </div>
  {#if data.next}<p><a href={`?after=${encodeURIComponent(data.next)}${data.search ? `&q=${encodeURIComponent(data.search)}` : ''}`}>{t.loadMore}</a></p>{/if}
</main>

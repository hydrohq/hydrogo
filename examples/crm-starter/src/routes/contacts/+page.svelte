<script lang="ts">
  import { enhance } from '$app/forms';
  import type { SubmitFunction } from '@sveltejs/kit';
  import Avatar from '$lib/components/Avatar.svelte';
  import Empty from '$lib/components/Empty.svelte';
  import FormError from '$lib/components/FormError.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import { messages } from '$lib/i18n';
  import { nextSort, sortRows, type Dir } from '$lib/table';
  let { data, form } = $props();
  const t = $derived(messages[data.lang]);

  let open = $state(false);
  const submit: SubmitFunction = () => async ({ result, update }) => {
    await update();
    if (result.type === 'success') open = false;
  };

  type Contact = (typeof data.contacts)[number];
  const companyOf = (c: Contact) => c.expanded?.company as { id: string; name?: string } | undefined;
  let sort = $state<{ key: string; dir: Dir }>({ key: '', dir: 1 });
  const columns = $derived([
    { key: 'full_name', label: t.name },
    { key: 'email', label: t.email },
    { key: 'company', label: t.company },
    { key: 'phone', label: t.phone }
  ]);
  const rows = $derived(sort.key ? sortRows(data.contacts, (c) => (sort.key === 'company' ? companyOf(c)?.name : c[sort.key]), sort.dir, data.lang) : data.contacts);
</script>

<svelte:head><title>{t.contacts} · {t.appName}</title></svelte:head>

<div class="page">
  <div class="toolbar" style="padding:0;border:0">
    <form method="GET" class="search" role="search"><input name="q" value={data.search} type="search" placeholder={t.search} aria-label={t.search} /><button aria-label={t.search}><Icon name="search" size={18} /></button></form>
    <span style="flex:1"></span>
    <button class="primary add" onclick={() => (open = true)}><Icon name="plus" size={16} /> {t.newContact}</button>
  </div>
  {#if !open}<FormError {form} lang={data.lang} />{/if}

  <article class="card tablecard">
    {#if rows.length === 0}
      <Empty icon="users" title={data.search ? t.noResults : t.empty} text={data.search ? t.tryAnother : t.addFirst} />
    {:else}
      <div class="scroll-x">
        <table class="data">
          <thead><tr>
            {#each columns as c (c.key)}
              <th scope="col" class:sorted={sort.key === c.key} aria-sort={sort.key === c.key ? (sort.dir === 1 ? 'ascending' : 'descending') : 'none'}>
                <button class="thbtn" onclick={() => (sort = nextSort(sort, c.key))}>{c.label}<span class="arrow" aria-hidden="true">{sort.key === c.key && sort.dir === -1 ? '↓' : '↑'}</span></button>
              </th>
            {/each}
          </tr></thead>
          <tbody>
            {#each rows as contact (contact.id)}
              {@const company = companyOf(contact)}
              <tr>
                <td class="namecell"><div class="who"><Avatar name={String(contact.full_name)} /><span>{contact.full_name}</span></div></td>
                <td>{contact.email ?? ''}</td>
                <td>{#if company}<a href={`/companies/${company.id}`}>{company.name}</a>{/if}</td>
                <td>{contact.phone ?? ''}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
    {#if data.next}
      <div class="table-foot"><span></span><a href={`?after=${encodeURIComponent(data.next)}${data.search ? `&q=${encodeURIComponent(data.search)}` : ''}`}>{t.loadMore} →</a></div>
    {/if}
  </article>
</div>

{#if open}
  <Modal title={t.newContact} closeLabel={t.close} onclose={() => (open = false)}>
    <form id="contact-form" method="POST" action="?/create" use:enhance={submit} class="formgrid">
      <label class="wide">{t.name}<input name="full_name" required /></label>
      <label>{t.email}<input name="email" type="email" /></label>
      <label>{t.company}<select name="company"><option value=""></option>{#each data.companies as company (company.id)}<option value={company.id}>{company.name}</option>{/each}</select></label>
      {#if form?.error}<div class="wide"><FormError {form} lang={data.lang} /></div>{/if}
    </form>
    {#snippet footer()}
      <button type="button" onclick={() => (open = false)}>{t.cancel}</button>
      <button class="primary" form="contact-form">{t.save}</button>
    {/snippet}
  </Modal>
{/if}

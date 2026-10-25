<script lang="ts">
  import { enhance } from '$app/forms';
  import type { SubmitFunction } from '@sveltejs/kit';
  import { messages, type Lang } from '$lib/i18n';
  import FormError from './FormError.svelte';
  import Icon from './Icon.svelte';
  import Modal from './Modal.svelte';
  let { lang, form = null }: { lang: Lang; form?: { error?: string; message?: string; requestId?: string; fields?: Record<string, string> } | null } = $props();
  const t = $derived(messages[lang]);
  let open = $state(false);
  const submit: SubmitFunction = () => async ({ result, update }) => {
    await update();
    if (result.type === 'success') open = false;
  };
</script>

<button class="primary add" onclick={() => (open = true)}><Icon name="plus" size={16} /> {t.newActivity}</button>

{#if open}
  <Modal title={t.newActivity} closeLabel={t.close} onclose={() => (open = false)}>
    <form id="activity-form" method="POST" action="?/activity" use:enhance={submit} class="formgrid">
      <label class="wide">{t.subject}<input name="subject" required /></label>
      <label>{t.kind}<select name="kind">{#each Object.entries(t.kinds) as [value, label] (value)}<option {value}>{label}</option>{/each}</select></label>
      <label>{t.due}<input name="due" type="datetime-local" /></label>
      {#if form?.error}<div class="wide"><FormError {form} {lang} /></div>{/if}
    </form>
    {#snippet footer()}
      <button type="button" onclick={() => (open = false)}>{t.cancel}</button>
      <button class="primary" form="activity-form">{t.save}</button>
    {/snippet}
  </Modal>
{/if}

<script lang="ts">
  import { enhance } from '$app/forms';
  import { messages, type Lang } from '$lib/i18n';
  import { isOverdue } from '$lib/pipeline';
  import Empty from './Empty.svelte';
  import Icon from './Icon.svelte';
  import { fly } from 'svelte/transition';
  import { ease, ms } from '$lib/motion';
  type Activity = { id: string; subject?: unknown; kind?: unknown; due?: unknown; done?: unknown };
  let { items, lang }: { items: Activity[]; lang: Lang } = $props();
  const t = $derived(messages[lang]);
  const icon = (kind: unknown) => (kind === 'call' ? 'phone' : kind === 'email' ? 'mail' : kind === 'meeting' ? 'calendar' : kind === 'note' ? 'note' : 'tasks');
</script>

{#if items.length === 0}
  <Empty icon="note" title={t.empty} />
{:else}
  <ul class="timeline">
    {#each items as item, i (item.id)}
      <li in:fly|local={{ y: 10, duration: ms(260), delay: ms(Math.min(i, 8) * 30), easing: ease }}>
        <span class="kindic"><Icon name={icon(item.kind)} size={16} /></span>
        <div class="body">
          <strong class:done={item.done}>{item.subject}</strong>
          <span class="muted small">{t.kinds[item.kind as keyof typeof t.kinds] ?? item.kind}{#if item.due} · <span class:overdue={!item.done && isOverdue(item.due)}>{new Date(item.due as string).toLocaleString(lang, { dateStyle: 'medium', timeStyle: 'short' })}</span>{/if}</span>
        </div>
        {#if !item.done}
          <form method="POST" action="?/done" use:enhance><input type="hidden" name="id" value={item.id} /><button class="check" aria-label={t.markDone} title={t.markDone}><Icon name="check" size={16} /></button></form>
        {:else}
          <span class="badge won">{t.done}</span>
        {/if}
      </li>
    {/each}
  </ul>
{/if}

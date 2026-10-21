<script lang="ts">
  import { enhance } from '$app/forms';
  import { messages, type Lang } from '$lib/i18n';
  let { lang }: { lang: Lang } = $props();
  const t = $derived(messages[lang]);
</script>

<form method="POST" action="?/activity" class="inline" use:enhance>
  <label>{t.subject}<input name="subject" required /></label>
  <label>{t.kind}<select name="kind">{#each Object.entries(t.kinds) as [value, label]}<option {value}>{label}</option>{/each}</select></label>
  <label>{t.due}<input name="due" type="datetime-local" /></label>
  <button class="primary">{t.newActivity}</button>
</form>

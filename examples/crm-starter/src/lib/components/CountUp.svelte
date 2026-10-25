<script lang="ts">
  import { Tween } from 'svelte/motion';
  import { ease, ms } from '$lib/motion';

  let { value, suffix = '' }: { value: number; suffix?: string } = $props();
  // Starts at the value on the server so the first paint is already right.
  // svelte-ignore state_referenced_locally
  const t = new Tween(value, { duration: ms(650), easing: ease });
  $effect(() => { t.target = value; });
</script>

<span style="font-variant-numeric: tabular-nums">{Math.round(t.current)}{suffix}</span>

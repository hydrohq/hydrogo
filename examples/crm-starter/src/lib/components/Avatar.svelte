<script lang="ts">
  import { initials, colorFor } from '$lib/avatar';
  let { name, size = 'md' }: { name: string; size?: 'md' | 'lg' | 'nav' } = $props();
</script>

<span class={`avatar ${size === 'md' ? '' : size}`} style={`background:${colorFor(name)}`}>{initials(name)}</span>

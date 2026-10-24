<script lang="ts">
  import { initials, colorFor } from '$lib/avatar';
  let { name, size = 'md' }: { name: string; size?: 'md' | 'lg' } = $props();
</script>

<span class={`avatar ${size === 'lg' ? 'lg' : ''}`} style={`background:${colorFor(name)}`}>{initials(name)}</span>

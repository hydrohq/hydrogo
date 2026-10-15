<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    description?: string;
    class?: string;
    headerAction?: Snippet;
    children?: Snippet;
  }

  let {
    title,
    description,
    class: className = '',
    headerAction,
    children,
  }: Props = $props();
</script>

<div class="rounded-xl border border-slate-800/90 bg-slate-900/60 backdrop-blur shadow-sm p-5 flex flex-col gap-4 {className}">
  {#if title || headerAction}
    <div class="flex items-center justify-between pb-3 border-b border-slate-800/60">
      <div>
        {#if title}
          <h3 class="text-base font-semibold text-slate-100">{title}</h3>
        {/if}
        {#if description}
          <p class="text-xs text-slate-400 mt-0.5">{description}</p>
        {/if}
      </div>
      {#if headerAction}
        <div>
          {@render headerAction()}
        </div>
      {/if}
    </div>
  {/if}
  {#if children}
    <div>
      {@render children()}
    </div>
  {/if}
</div>

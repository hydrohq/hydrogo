<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    description?: string;
    class?: string;
    headerAction?: Snippet;
    children?: Snippet;
    footer?: Snippet;
  }

  let {
    title,
    description,
    class: className = '',
    headerAction,
    children,
    footer,
  }: Props = $props();
</script>

<div class="rounded-xl border border-border bg-card text-card-foreground shadow-sm p-5 flex flex-col gap-4 {className}">
  {#if title || description || headerAction}
    <div class="flex items-center justify-between pb-3 border-b border-border/60">
      <div class="space-y-0.5">
        {#if title}
          <h3 class="text-base font-semibold tracking-tight text-foreground">{title}</h3>
        {/if}
        {#if description}
          <p class="text-xs text-muted-foreground">{description}</p>
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
    <div class="flex-1">
      {@render children()}
    </div>
  {/if}

  {#if footer}
    <div class="pt-3 border-t border-border/60 flex items-center justify-between text-xs text-muted-foreground">
      {@render footer()}
    </div>
  {/if}
</div>

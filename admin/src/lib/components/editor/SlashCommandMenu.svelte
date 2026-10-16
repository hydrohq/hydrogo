<script lang="ts">
  export interface SlashItem {
    id: string;
    title: string;
    description: string;
    badge?: string;
    action: () => void;
  }

  interface Props {
    items: SlashItem[];
    selectedIndex: number;
    position: { top: number; left: number };
    onSelect: (item: SlashItem) => void;
    onHover: (index: number) => void;
  }

  let { items, selectedIndex, position, onSelect, onHover }: Props = $props();
</script>

<div
  class="absolute z-40 w-72 max-h-72 overflow-y-auto rounded-xl border border-border bg-popover/95 backdrop-blur-md p-1.5 text-popover-foreground shadow-2xl animate-in fade-in zoom-in-95 duration-100"
  style="top: {position.top}px; left: {position.left}px;"
>
  <div class="px-2 py-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/50 mb-1">
    Basic Blocks
  </div>

  <div class="space-y-0.5">
    {#each items as cmd, i (cmd.id)}
      <button
        type="button"
        class="w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-left transition-colors cursor-pointer {i === selectedIndex ? 'bg-accent text-accent-foreground font-medium' : 'text-foreground hover:bg-muted/50'}"
        onclick={() => onSelect(cmd)}
        onmouseenter={() => onHover(i)}
      >
        <!-- Icon badge -->
        <div class="size-7 rounded-md bg-muted border border-border flex items-center justify-center text-xs font-mono font-semibold shrink-0">
          {cmd.badge}
        </div>

        <!-- Text info -->
        <div class="flex-1 min-w-0">
          <div class="text-xs font-medium leading-none truncate">
            {cmd.title}
          </div>
          <div class="text-[10px] text-muted-foreground truncate mt-0.5">
            {cmd.description}
          </div>
        </div>
      </button>
    {/each}
  </div>
</div>

<script lang="ts">
  // Code-splits a route view: the chunk for `load` downloads only once this
  // component is actually rendered, instead of being bundled into the app shell.
  type Props = {
    load: () => Promise<{ default: unknown }>;
    props?: Record<string, unknown>;
  };

  let { load, props = {} }: Props = $props();
</script>

{#await load()}
  <div class="flex items-center justify-center py-16">
    <div class="size-6 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
  </div>
{:then mod}
  {@const Component = mod.default as any}
  <Component {...props} />
{:catch error}
  <div class="p-8 max-w-2xl mx-auto rounded-xl border border-destructive/30 bg-destructive/5 text-center space-y-2">
    <p class="text-sm font-medium text-destructive">Failed to load view</p>
    <p class="text-xs text-muted-foreground">{error instanceof Error ? error.message : String(error)}</p>
  </div>
{/await}

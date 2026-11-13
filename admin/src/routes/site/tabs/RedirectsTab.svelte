<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import Input from '$lib/components/ui/Input.svelte';
  import Badge from '$lib/components/ui/Badge.svelte';

  export interface RedirectRuleRow {
    id: string;
    pattern?: string;
    source?: string;
    kind?: string;
    target_path?: string;
    target?: string;
    status?: number;
    code?: number;
    hits?: number;
  }

  let {
    redirects,
    newPattern = $bindable(''),
    newTarget = $bindable(''),
    newKind = $bindable('exact'),
    newStatus = $bindable(301),
    saving = false,
    oncreate,
    ondelete,
  }: {
    redirects: RedirectRuleRow[];
    newPattern: string;
    newTarget: string;
    newKind: string;
    newStatus: number;
    saving: boolean;
    oncreate: (e: SubmitEvent) => void;
    ondelete: (rule: RedirectRuleRow) => void;
  } = $props();
</script>

<div class="space-y-6">
  <form onsubmit={oncreate} class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
    <h3 class="text-sm font-semibold text-foreground">{t('site.redir.add')}</h3>
    <div class="grid grid-cols-1 md:grid-cols-12 gap-3 items-end">
      <div class="md:col-span-4">
        <label for="red-from" class="block text-xs font-medium text-foreground mb-1">{t('site.redir.from')}</label>
        <Input id="red-from" type="text" bind:value={newPattern} placeholder="/eski-urun" required />
      </div>
      <div class="md:col-span-4">
        <label for="red-to" class="block text-xs font-medium text-foreground mb-1">{t('site.redir.to')}</label>
        <Input id="red-to" type="text" bind:value={newTarget} placeholder="/urunler" required />
      </div>
      <div class="md:col-span-2">
        <label for="red-kind" class="block text-xs font-medium text-foreground mb-1">{t('site.redir.kind')}</label>
        <select
          id="red-kind"
          bind:value={newKind}
          class="h-9 w-full rounded-md border border-input bg-background px-2.5 text-xs text-foreground"
        >
          <option value="exact">exact</option>
          <option value="wildcard">wildcard (*)</option>
          <option value="regex">regex</option>
        </select>
      </div>
      <div class="md:col-span-1">
        <label for="red-status" class="block text-xs font-medium text-foreground mb-1">{t('site.redir.status')}</label>
        <select
          id="red-status"
          bind:value={newStatus}
          class="h-9 w-full rounded-md border border-input bg-background px-2 text-xs text-foreground"
        >
          <option value={301}>301</option>
          <option value={302}>302</option>
          <option value={308}>308</option>
        </select>
      </div>
      <div class="md:col-span-1">
        <Button type="submit" class="w-full" disabled={saving}>{t('site.redir.addRule')}</Button>
      </div>
    </div>
  </form>

  <div class="rounded-xl border border-border bg-card overflow-hidden shadow-xs">
    <div class="p-4 border-b border-border flex items-center justify-between">
      <h3 class="text-sm font-semibold text-foreground">{t('site.redir.active', { count: redirects.length })}</h3>
    </div>
    <div class="divide-y divide-border">
      {#if redirects.length === 0}
        <div class="p-8 text-center text-xs text-muted-foreground">{t('site.redir.empty')}</div>
      {:else}
        {#each redirects as r (r.id)}
          <div class="p-4 flex items-center justify-between gap-4 hover:bg-muted/30 transition-colors">
            <div class="grid text-left min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-mono text-xs font-semibold text-foreground">{r.pattern || r.source}</span>
                <span class="text-muted-foreground text-xs">→</span>
                <span class="font-mono text-xs text-primary">{r.target_path || r.target}</span>
                <Badge variant="outline" class="text-[10px]">{r.status || r.code || 301}</Badge>
                {#if r.kind}<Badge variant="outline" class="text-[10px]">{r.kind}</Badge>{/if}
              </div>
            </div>
            <Button variant="danger" onclick={() => ondelete(r)}>{t('common.delete')}</Button>
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>

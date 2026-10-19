<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import { createPresence, type PresenceState } from '$lib/presence.svelte';

  interface Props {
    presence?: PresenceState;
    endpoint?: string;
    resourceId?: string | null;
    mode?: 'pill' | 'banner' | 'all';
    class?: string;
  }

  let {
    presence: externalPresence,
    endpoint = '',
    resourceId = null,
    mode = 'all',
    class: className = '',
  }: Props = $props();

  const internalPresence = createPresence(
    () => (externalPresence ? '' : endpoint),
    () => (externalPresence ? null : resourceId)
  );

  const currentPresence = $derived(externalPresence ?? internalPresence);
  let showPopover = $state(false);

  const COLOR_PALETTES = [
    { bg: 'bg-emerald-500/15', text: 'text-emerald-700 dark:text-emerald-300', border: 'border-emerald-500/30' },
    { bg: 'bg-indigo-500/15', text: 'text-indigo-700 dark:text-indigo-300', border: 'border-indigo-500/30' },
    { bg: 'bg-violet-500/15', text: 'text-violet-700 dark:text-violet-300', border: 'border-violet-500/30' },
    { bg: 'bg-amber-500/15', text: 'text-amber-700 dark:text-amber-300', border: 'border-amber-500/30' },
    { bg: 'bg-rose-500/15', text: 'text-rose-700 dark:text-rose-300', border: 'border-rose-500/30' },
    { bg: 'bg-sky-500/15', text: 'text-sky-700 dark:text-sky-300', border: 'border-sky-500/30' },
    { bg: 'bg-teal-500/15', text: 'text-teal-700 dark:text-teal-300', border: 'border-teal-500/30' },
    { bg: 'bg-fuchsia-500/15', text: 'text-fuchsia-700 dark:text-fuchsia-300', border: 'border-fuchsia-500/30' },
  ];

  function getPalette(nameOrEmail: string) {
    let hash = 0;
    for (let i = 0; i < nameOrEmail.length; i++) {
      hash = (hash << 5) - hash + nameOrEmail.charCodeAt(i);
      hash |= 0;
    }
    const idx = Math.abs(hash) % COLOR_PALETTES.length;
    return COLOR_PALETTES[idx];
  }

  function getInitials(nameOrEmail: string): string {
    const clean = nameOrEmail.split('@')[0].trim();
    const parts = clean.split(/[ ._-]+/).filter(Boolean);
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase();
    }
    return (clean.slice(0, 2) || 'ED').toUpperCase();
  }

  function formatActiveTime(secs: number): string {
    if (secs <= 2) return 'Şu anda aktif';
    if (secs < 60) return `${secs} sn önce`;
    const mins = Math.floor(secs / 60);
    return `${mins} dk önce`;
  }
</script>

{#if currentPresence.count > 0}
  <!-- Inline Top Bar Presence Pill -->
  {#if mode === 'pill' || mode === 'all'}
    <div class="relative inline-flex items-center {className}">
      <button
        type="button"
        onclick={() => { showPopover = !showPopover; }}
        class="group inline-flex items-center gap-2 px-2.5 py-1 rounded-full border border-amber-500/30 bg-amber-500/10 hover:bg-amber-500/15 text-xs text-amber-800 dark:text-amber-200 transition-all cursor-pointer shadow-xs animate-fade-in"
        title={t('presence.pillTitle')}
      >
        <!-- Pulsing live indicator -->
        <span class="relative flex size-2 shrink-0">
          <span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75"></span>
          <span class="relative inline-flex size-2 rounded-full bg-amber-500"></span>
        </span>

        <!-- Avatars Stack -->
        <div class="flex -space-x-1.5 overflow-hidden py-0.5">
          {#each currentPresence.activeEditors.slice(0, 3) as editor (editor.user_id || editor.email || editor.name)}
            {@const palette = getPalette(editor.email || editor.name)}
            <span
              class="inline-flex size-5 items-center justify-center rounded-full {palette.bg} {palette.text} border {palette.border} text-[10px] font-bold shadow-xs shrink-0 select-none"
              title="{editor.name} ({editor.email || ''})"
            >
              {getInitials(editor.name || editor.email || 'ED')}
            </span>
          {/each}
          {#if currentPresence.count > 3}
            <span
              class="inline-flex size-5 items-center justify-center rounded-full bg-muted border border-border text-[9px] font-semibold text-muted-foreground shrink-0 select-none"
            >
              +{currentPresence.count - 3}
            </span>
          {/if}
        </div>

        <!-- Text Label -->
        <span class="font-medium text-[11px] hidden sm:inline-block">
          {#if currentPresence.count === 1}
            <span class="font-semibold">{currentPresence.activeEditors[0].name}</span> {t('presence.isEditing')}
          {:else}
            <span class="font-semibold">{t('presence.people', { count: currentPresence.count })}</span> {t('presence.isEditing')}
          {/if}
        </span>
      </button>

      <!-- Dropdown / Popover Details -->
      {#if showPopover}
        <div
          class="absolute left-0 top-full mt-1.5 w-64 rounded-lg border border-border bg-popover p-2.5 text-popover-foreground shadow-lg z-50 animate-modal-in"
        >
          <div class="flex items-center justify-between pb-1.5 mb-1.5 border-b border-border text-[11px] font-semibold text-muted-foreground">
            <span>{t('presence.popoverTitle')}</span>
            <button
              type="button"
              onclick={() => { showPopover = false; }}
              class="hover:text-foreground text-xs p-0.5 rounded cursor-pointer"
            >
              ✕
            </button>
          </div>
          <div class="flex flex-col gap-1.5 max-h-48 overflow-y-auto">
            {#each currentPresence.activeEditors as editor (editor.user_id || editor.email || editor.name)}
              {@const palette = getPalette(editor.email || editor.name)}
              <div class="flex items-center justify-between p-1 rounded hover:bg-muted/50 transition-colors">
                <div class="flex items-center gap-2 min-w-0">
                  <span
                    class="inline-flex size-6 items-center justify-center rounded-full {palette.bg} {palette.text} border {palette.border} text-[10px] font-bold shrink-0 select-none"
                  >
                    {getInitials(editor.name || editor.email || 'ED')}
                  </span>
                  <div class="flex flex-col min-w-0">
                    <span class="text-xs font-medium text-foreground truncate">{editor.name}</span>
                    {#if editor.email}
                      <span class="text-[10px] text-muted-foreground truncate">{editor.email}</span>
                    {/if}
                  </div>
                </div>
                <span class="text-[10px] text-muted-foreground whitespace-nowrap ml-2">
                  {formatActiveTime(editor.active_seconds)}
                </span>
              </div>
            {/each}
          </div>
          <p class="mt-2 text-[10px] text-amber-600 dark:text-amber-400 bg-amber-500/10 p-1.5 rounded leading-tight">
            ⚠️ Aynı anda yapılan değişiklikler son kaydedenin verisinin geçerli olmasına (çakışmaya) neden olabilir.
          </p>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Prominent Non-blocking Warning Banner -->
  {#if (mode === 'banner' || mode === 'all') && !currentPresence.dismissedBanner}
    <div
      class="mx-4 lg:mx-6 p-3 rounded-lg bg-amber-500/10 border border-amber-500/30 text-amber-800 dark:text-amber-200 text-xs flex items-center justify-between gap-3 animate-fade-in shadow-xs {className}"
    >
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="size-6 rounded-full bg-amber-500/20 text-amber-600 dark:text-amber-400 flex items-center justify-center shrink-0">
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
        </div>
        <div class="leading-relaxed">
          <strong>{t('presence.warnTitle')}</strong>
          {t('presence.warnBody', { names: currentPresence.names })}
        </div>
      </div>
      <button
        type="button"
        onclick={() => { currentPresence.dismiss(); }}
        class="text-amber-700 hover:text-amber-950 dark:text-amber-300 dark:hover:text-amber-100 p-1 rounded hover:bg-amber-500/20 transition-colors cursor-pointer shrink-0"
        title={t('presence.dismiss')}
      >
        <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </button>
    </div>
  {/if}
{/if}

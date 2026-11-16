<script lang="ts">
  // Grouped content-type sections for the admin sidebar (split from Sidebar.svelte to stay concise).
  import { t } from '$lib/i18n/index.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';
  import type { Collection } from '$lib/api/types';
  import {
    isAdvancedGroup,
    isDefaultContentGroup,
    type CollectionGroup,
  } from './sidebarGroups';

  interface Props {
    collectionGroups: CollectionGroup<Collection>[];
    canWriteContent: boolean;
    activeDocDropdown: string | null;
    isGroupCollapsed: (groupName: string) => boolean;
    toggleGroup: (groupName: string) => void;
    isActive: (target: string) => boolean;
    handleNavigate: (path: string) => void;
    onToggleDocDropdown: (id: string) => void;
  }

  let {
    collectionGroups,
    canWriteContent,
    activeDocDropdown,
    isGroupCollapsed,
    toggleGroup,
    isActive,
    handleNavigate,
    onToggleDocDropdown,
  }: Props = $props();

  const schemaGroupName = $derived.by(() => {
    const def = collectionGroups.find((g) => isDefaultContentGroup(g.name));
    if (def) return def.name;
    const nonAdv = collectionGroups.find((g) => !isAdvancedGroup(g.name));
    return nonAdv?.name ?? collectionGroups[0]?.name ?? '';
  });
</script>

{#each collectionGroups as group (group.name)}
  {@const isDefault = isDefaultContentGroup(group.name)}
  {@const isAdvanced = isAdvancedGroup(group.name)}
  {@const showSchemaButton = canWriteContent && group.name === schemaGroupName}
  {@const groupLabel = isDefault ? t('nav.documents') : isAdvanced ? t('nav.advanced') : group.name}
  {@const collapsed = sidebar.open && !isDefault && isGroupCollapsed(group.name)}

  <div class="mt-4 flex flex-col gap-0.5">
    {#if sidebar.open}
      {#if isDefault}
        <div class="flex h-7 items-center justify-between px-2.5 text-xs font-medium text-sidebar-foreground/60">
          <span>{groupLabel}</span>
          {#if showSchemaButton}
            <button
              type="button"
              onclick={() => handleNavigate('/content-types')}
              class="flex size-5 items-center justify-center rounded text-sidebar-foreground/60 transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground cursor-pointer"
              title={t('nav.contentTypes')}
              aria-label={t('nav.contentTypes')}
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
            </button>
          {/if}
        </div>
      {:else}
        <div class="flex h-7 items-center justify-between px-2.5">
          <button
            type="button"
            onclick={() => toggleGroup(group.name)}
            aria-expanded={!collapsed}
            data-testid="nav-group-{group.name}"
            class="flex flex-1 items-center justify-between text-left text-xs font-medium text-sidebar-foreground/60 transition-colors hover:text-sidebar-foreground cursor-pointer"
          >
            <span>{groupLabel}</span>
            <svg class="size-3.5 transition-transform duration-150 {collapsed ? '-rotate-90' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="6 9 12 15 18 9"></polyline>
            </svg>
          </button>
          {#if showSchemaButton}
            <button
              type="button"
              onclick={() => handleNavigate('/content-types')}
              class="ml-1 flex size-5 items-center justify-center rounded text-sidebar-foreground/60 transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground cursor-pointer"
              title={t('nav.contentTypes')}
              aria-label={t('nav.contentTypes')}
            >
              <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <line x1="12" y1="5" x2="12" y2="19"></line>
                <line x1="5" y1="12" x2="19" y2="12"></line>
              </svg>
            </button>
          {/if}
        </div>
      {/if}
    {/if}

    {#if !sidebar.open || isDefault || !collapsed}
      <div class="flex flex-col gap-0.5">
        {#each group.collections as collection (collection.id)}
          {@const isSingleton = collection.type === 'global'}
          <div class="relative group flex items-center" data-dropdown="doc-{collection.id}">
            <a
              href="/admin/content/{collection.id}"
              onclick={() => handleNavigate(`/content/${collection.id}`)}
              class="flex h-8 flex-1 items-center gap-2.5 rounded-md px-2.5 text-sm transition-colors duration-150 {isActive(`/content/${collection.id}`) ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/75 hover:bg-sidebar-accent/70 hover:text-sidebar-accent-foreground'}"
              title={collection.plural || collection.label}
            >
              <svg class="size-4 shrink-0 text-sidebar-foreground/60" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                {#if isSingleton}
                  <rect width="18" height="18" x="3" y="3" rx="2"></rect>
                  <path d="M7 8h10"></path>
                  <path d="M7 12h6"></path>
                  <path d="M7 16h8"></path>
                {:else}
                  <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z"></path>
                  <polyline points="14 2 14 8 20 8"></polyline>
                  <line x1="16" y1="13" x2="8" y2="13"></line>
                  <line x1="16" y1="17" x2="8" y2="17"></line>
                {/if}
              </svg>
              {#if sidebar.open}
                <span class="truncate">{collection.plural || collection.label}</span>
              {/if}
            </a>

            {#if sidebar.open && !isSingleton}
              <button
                type="button"
                onclick={(e) => {
                  e.stopPropagation();
                  onToggleDocDropdown(collection.id);
                }}
                class="absolute right-1 flex size-6 items-center justify-center rounded-sm opacity-0 group-hover:opacity-100 hover:bg-sidebar-accent text-sidebar-foreground/60 hover:text-sidebar-foreground transition-opacity duration-150 cursor-pointer"
                title={t('nav.options')}
              >
                <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <circle cx="12" cy="12" r="1"></circle>
                  <circle cx="19" cy="12" r="1"></circle>
                  <circle cx="5" cy="12" r="1"></circle>
                </svg>
              </button>

              {#if activeDocDropdown === collection.id}
                <div class="absolute right-0 top-8 z-50 w-32 rounded-lg border border-border bg-popover p-1 text-xs text-popover-foreground shadow-md animate-modal-in">
                  <button
                    type="button"
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-accent cursor-pointer"
                    onclick={() => handleNavigate(`/content/${collection.id}`)}
                  >
                    <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2Z"></path></svg>
                    <span>{t('nav.open')}</span>
                  </button>
                  <button
                    type="button"
                    class="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-accent cursor-pointer"
                    onclick={() => handleNavigate(`/content/${collection.id}/new`)}
                  >
                    <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="12" y1="5" x2="12" y2="19"></line><line x1="5" y1="12" x2="19" y2="12"></line></svg>
                    <span>{t('nav.create')}</span>
                  </button>
                </div>
              {/if}
            {/if}
          </div>
        {/each}
      </div>
    {/if}
  </div>
{/each}

<script lang="ts">
  // Store and customer navigation groups of the admin sidebar (split out of Sidebar.svelte to keep it under the file-size budget).
  import { t } from '$lib/i18n/index.svelte';
  import { schema } from '$lib/stores/schema.svelte';
  import { sidebar } from '$lib/stores/sidebar.svelte';

  let { moduleOn, can, isActive, handleNavigate, unresolvedRefundCount, pendingReviews }: {
    moduleOn: (name: string) => boolean;
    can: (permission: string) => boolean;
    isActive: (target: string) => boolean;
    handleNavigate: (path: string) => void;
    unresolvedRefundCount: number;
    pendingReviews: number;
  } = $props();
</script>

<!-- Store Navigation -->
{#if moduleOn('commerce') && can('admin.commerce.read')}
<div class="mt-2 flex flex-col gap-0.5">
  {#if sidebar.open}
    <div class="px-2 py-1 text-xs font-medium tracking-wider text-sidebar-foreground/60">{t('nav.store')}</div>
  {/if}
  <!-- Products -->
  <a
    href="/admin/commerce/products"
    onclick={() => handleNavigate('/commerce/products')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/commerce/products') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.products')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M12 2H2v10l9.29 9.29c.94.94 2.48.94 3.42 0l6.58-6.58c.94-.94.94-2.48 0-3.42L12 2Z"></path>
      <path d="M7 7h.01"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.products')}</span>
    {/if}
  </a>

  {#if schema.getCollection('categories')}
  <!-- Categories -->
  <a
    href="/admin/content/categories"
    onclick={() => handleNavigate('/content/categories')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/content/categories') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.categories')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.categories')}</span>
    {/if}
  </a>
  {/if}

  <!-- Commerce orders -->
  <a
    href="/admin/commerce"
    onclick={() => handleNavigate('/commerce')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/commerce') && !isActive('/commerce/refunds-required') && !isActive('/commerce/discount-codes') && !isActive('/commerce/stock') && !isActive('/commerce/products') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.orders')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M6 2 3 6v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V6l-3-4Z"></path>
      <path d="M3 6h18"></path>
      <path d="M16 10a4 4 0 0 1-8 0"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.orders')}</span>
    {/if}
  </a>

  {#if schema.getCollection('return_requests')}
  <!-- Return requests -->
  <a
    href="/admin/content/return_requests"
    onclick={() => handleNavigate('/content/return_requests')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/content/return_requests') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.returnRequests')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M9 14 4 9l5-5"></path>
      <path d="M4 9h10.5a5.5 5.5 0 0 1 5.5 5.5a5.5 5.5 0 0 1-5.5 5.5H11"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.returnRequests')}</span>
    {/if}
  </a>
  {/if}

  <!-- Stock -->
  <a
    href="/admin/commerce/stock"
    onclick={() => handleNavigate('/commerce/stock')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/commerce/stock') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.stock')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16Z"></path>
      <path d="m3.3 7 8.7 5 8.7-5"></path>
      <path d="M12 22V12"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.stock')}</span>
    {/if}
  </a>

  <!-- Discount codes -->
  <a
    href="/admin/commerce/discount-codes"
    onclick={() => handleNavigate('/commerce/discount-codes')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/commerce/discount-codes') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.discountCodes')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0L3 13V3h10l7.6 7.6a2 2 0 0 1 0 2.8z"></path>
      <circle cx="7.5" cy="7.5" r="1.5"></circle>
    </svg>
    {#if sidebar.open}<span class="truncate">{t('nav.discountCodes')}</span>{/if}
  </a>

  <!-- Refunds required -->
  <a
    href="/admin/commerce/refunds-required"
    onclick={() => handleNavigate('/commerce/refunds-required')}
    class="flex items-center justify-between gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/commerce/refunds-required') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.refundsRequired')}
  >
    <div class="flex items-center gap-2 min-w-0">
      <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"></path>
        <path d="M3 3v5h5"></path>
      </svg>
      {#if sidebar.open}
        <span class="truncate">{t('nav.refundsRequired')}</span>
      {/if}
    </div>
    {#if unresolvedRefundCount > 0}
      <span class="shrink-0 rounded-full bg-destructive/10 px-1.5 py-0.5 text-[10px] font-medium text-destructive">{unresolvedRefundCount}</span>
    {/if}
  </a>
</div>
{/if}

<!-- Customers & Community -->
{#if (moduleOn('members') && can('admin.members.manage')) || schema.getCollection('reviews') || schema.getCollection('leads')}
<div class="mt-2 flex flex-col gap-0.5">
  {#if sidebar.open}
    <div class="px-2 py-1 text-xs font-medium tracking-wider text-sidebar-foreground/60">{t('nav.customers')}</div>
  {/if}

  {#if moduleOn('members') && can('admin.members.manage')}
  <!-- Members -->
  <a
    href="/admin/members"
    onclick={() => handleNavigate('/members')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/members') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.members')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"></path>
      <circle cx="9" cy="7" r="4"></circle>
      <path d="M22 21v-2a4 4 0 0 0-3-3.87"></path>
      <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.members')}</span>
    {/if}
  </a>
  {/if}

  {#if schema.getCollection('reviews')}
  <!-- Reviews -->
  <a
    href="/admin/content/reviews"
    onclick={() => handleNavigate('/content/reviews')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/content/reviews') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.reviews')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.reviews')}</span>
    {/if}
    {#if pendingReviews > 0}
      <span class="ml-auto shrink-0 rounded-full bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-medium text-amber-600" title={t('common.draft')}>{pendingReviews >= 100 ? '99+' : pendingReviews}</span>
    {/if}
  </a>
  {/if}

  {#if schema.getCollection('leads')}
  <!-- Leads -->
  <a
    href="/admin/content/leads"
    onclick={() => handleNavigate('/content/leads')}
    class="flex items-center gap-2 rounded-md px-2.5 py-1.5 text-sm transition-all duration-150 {isActive('/content/leads') ? 'bg-sidebar-accent text-sidebar-accent-foreground font-medium shadow-xs' : 'text-sidebar-foreground/80 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground hover:translate-x-0.5'}"
    title={t('nav.leads')}
  >
    <svg class="size-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <polyline points="22 12 16 12 14 15 10 15 8 12 2 12"></polyline>
      <path d="M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"></path>
    </svg>
    {#if sidebar.open}
      <span class="truncate">{t('nav.leads')}</span>
    {/if}
  </a>
  {/if}
</div>
{/if}

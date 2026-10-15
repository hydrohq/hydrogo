<script lang="ts">
  import rawTableData from './data.json';

  interface TableRowItem {
    id: number;
    header: string;
    type: string;
    status: string;
    target: string;
    limit: string;
    reviewer: string;
  }

  let items = $state<TableRowItem[]>(rawTableData as TableRowItem[]);
  let activeTab = $state<'outline' | 'past-performance' | 'key-personnel' | 'focus-documents'>('outline');
  let selectedIds = $state<number[]>([]);
  let searchQuery = $state('');
  let pageIndex = $state(0);
  let pageSize = $state(10);
  let showColumnsMenu = $state(false);

  // Column visibility
  let visibleColumns = $state({
    header: true,
    type: true,
    status: true,
    target: true,
    limit: true,
    reviewer: true,
  });

  const filteredItems = $derived.by(() => {
    let list = items;
    if (activeTab === 'past-performance') {
      list = list.filter(i => i.id % 4 === 0);
    } else if (activeTab === 'key-personnel') {
      list = list.filter(i => i.reviewer !== 'Assign reviewer');
    } else if (activeTab === 'focus-documents') {
      list = list.filter(i => i.status === 'In Process');
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      list = list.filter(i =>
        i.header.toLowerCase().includes(q) ||
        i.type.toLowerCase().includes(q) ||
        i.reviewer.toLowerCase().includes(q)
      );
    }
    return list;
  });

  const totalPages = $derived(Math.max(1, Math.ceil(filteredItems.length / pageSize)));
  const paginatedItems = $derived(
    filteredItems.slice(pageIndex * pageSize, (pageIndex + 1) * pageSize)
  );

  const allSelected = $derived(
    paginatedItems.length > 0 && paginatedItems.every(i => selectedIds.includes(i.id))
  );

  function toggleSelectAll() {
    if (allSelected) {
      selectedIds = selectedIds.filter(id => !paginatedItems.some(i => i.id === id));
    } else {
      const newIds = new Set([...selectedIds, ...paginatedItems.map(i => i.id)]);
      selectedIds = Array.from(newIds);
    }
  }

  function toggleSelectRow(id: number) {
    if (selectedIds.includes(id)) {
      selectedIds = selectedIds.filter(i => i !== id);
    } else {
      selectedIds = [...selectedIds, id];
    }
  }
</script>

<div class="flex flex-col gap-4 px-4 lg:px-6">
  <!-- Tabs and Action Toolbar Header (Exact shadcn specification) -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
    <!-- TabsList -->
    <div class="flex items-center rounded-lg bg-muted p-1 text-xs border border-border overflow-x-auto">
      <button
        type="button"
        class="px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap {activeTab === 'outline' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'outline'; pageIndex = 0; }}
      >
        Outline
      </button>

      <button
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap {activeTab === 'past-performance' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'past-performance'; pageIndex = 0; }}
      >
        <span>Past Performance</span>
        <span class="flex h-4.5 w-4.5 items-center justify-center rounded-full bg-muted-foreground/20 text-[10px] font-mono">
          3
        </span>
      </button>

      <button
        type="button"
        class="flex items-center gap-1.5 px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap {activeTab === 'key-personnel' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'key-personnel'; pageIndex = 0; }}
      >
        <span>Key Personnel</span>
        <span class="flex h-4.5 w-4.5 items-center justify-center rounded-full bg-muted-foreground/20 text-[10px] font-mono">
          2
        </span>
      </button>

      <button
        type="button"
        class="px-3 py-1.5 rounded-md font-medium transition-all whitespace-nowrap {activeTab === 'focus-documents' ? 'bg-background text-foreground shadow-xs' : 'text-muted-foreground hover:text-foreground'}"
        onclick={() => { activeTab = 'focus-documents'; pageIndex = 0; }}
      >
        Focus Documents
      </button>
    </div>

    <!-- Actions & Filter Toolbar -->
    <div class="flex items-center gap-2">
      <!-- Search -->
      <div class="relative">
        <input
          type="search"
          placeholder="Filter sections..."
          bind:value={searchQuery}
          class="h-8 w-36 sm:w-56 rounded-md border border-input bg-background px-3 text-xs shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
        />
      </div>

      <!-- Customize Columns Dropdown -->
      <div class="relative">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 h-8 px-2.5 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium shadow-xs text-foreground cursor-pointer"
          onclick={() => { showColumnsMenu = !showColumnsMenu; }}
        >
          <svg class="size-3.5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 3h7a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-7m0-18H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h7m0-18v18"></path>
          </svg>
          <span class="hidden sm:inline">Customize Columns</span>
          <span class="sm:hidden">Columns</span>
          <svg class="size-3 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="6 9 12 15 18 9"></polyline>
          </svg>
        </button>

        {#if showColumnsMenu}
          <div class="absolute right-0 top-10 w-48 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-md z-30 text-xs">
            {#each Object.keys(visibleColumns) as col}
              <label class="flex items-center gap-2 px-2.5 py-1.5 rounded-sm hover:bg-accent cursor-pointer capitalize">
                <input
                  type="checkbox"
                  bind:checked={(visibleColumns as any)[col]}
                  class="rounded border-input text-primary focus:ring-ring size-3.5"
                />
                <span>{col}</span>
              </label>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Add Section Button -->
      <button
        type="button"
        class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium shadow-xs cursor-pointer"
        onclick={() => {
          items = [
            {
              id: Date.now(),
              header: "New Proposal Section",
              type: "Narrative",
              status: "In Process",
              target: "10",
              limit: "15",
              reviewer: "Assign reviewer"
            },
            ...items
          ];
        }}
      >
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
        <span>Add Section</span>
      </button>
    </div>
  </div>

  <!-- The DataTable Container -->
  <div class="overflow-hidden rounded-xl border border-border bg-card shadow-xs">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-sm text-foreground caption-bottom">
        <thead class="sticky top-0 z-10 bg-muted/50 border-b border-border text-xs text-muted-foreground font-medium">
          <tr>
            <!-- Drag Handle Column -->
            <th class="h-10 w-10 px-2 text-center font-medium"></th>

            <!-- Select Checkbox Column -->
            <th class="h-10 w-10 px-2 text-center font-medium">
              <input
                type="checkbox"
                checked={allSelected}
                onchange={toggleSelectAll}
                class="rounded border-input text-primary focus:ring-ring size-4 cursor-pointer"
                aria-label="Select all rows"
              />
            </th>

            {#if visibleColumns.header}
              <th class="h-10 px-4 font-medium">Header</th>
            {/if}

            {#if visibleColumns.type}
              <th class="h-10 px-4 font-medium">Section Type</th>
            {/if}

            {#if visibleColumns.status}
              <th class="h-10 px-4 font-medium">Status</th>
            {/if}

            {#if visibleColumns.target}
              <th class="h-10 px-4 font-medium text-right">Target</th>
            {/if}

            {#if visibleColumns.limit}
              <th class="h-10 px-4 font-medium text-right">Limit</th>
            {/if}

            {#if visibleColumns.reviewer}
              <th class="h-10 px-4 font-medium">Reviewer</th>
            {/if}

            <th class="h-10 w-12 px-2 text-center font-medium"></th>
          </tr>
        </thead>

        <tbody class="divide-y divide-border">
          {#if paginatedItems.length === 0}
            <tr>
              <td colspan="9" class="p-8 text-center text-sm text-muted-foreground">
                No results found.
              </td>
            </tr>
          {:else}
            {#each paginatedItems as row (row.id)}
              {@const isSelected = selectedIds.includes(row.id)}
              <tr
                class="hover:bg-muted/40 transition-colors {isSelected ? 'bg-muted/60' : ''}"
              >
                <!-- Drag Handle -->
                <td class="p-2 text-center text-muted-foreground/60 cursor-grab">
                  <svg class="size-3.5 inline-block" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="9" cy="5" r="1"></circle>
                    <circle cx="9" cy="12" r="1"></circle>
                    <circle cx="9" cy="19" r="1"></circle>
                    <circle cx="15" cy="5" r="1"></circle>
                    <circle cx="15" cy="12" r="1"></circle>
                    <circle cx="15" cy="19" r="1"></circle>
                  </svg>
                </td>

                <!-- Row Select Checkbox -->
                <td class="p-2 text-center">
                  <input
                    type="checkbox"
                    checked={isSelected}
                    onchange={() => toggleSelectRow(row.id)}
                    class="rounded border-input text-primary focus:ring-ring size-4 cursor-pointer"
                    aria-label={`Select row ${row.header}`}
                  />
                </td>

                <!-- Header Title -->
                {#if visibleColumns.header}
                  <td class="p-4 font-medium text-foreground max-w-xs truncate">
                    <span>{row.header}</span>
                  </td>
                {/if}

                <!-- Section Type Badge -->
                {#if visibleColumns.type}
                  <td class="p-4">
                    <span class="inline-flex items-center rounded-md border border-border px-2 py-0.5 text-xs text-muted-foreground font-normal bg-background">
                      {row.type}
                    </span>
                  </td>
                {/if}

                <!-- Status Badge -->
                {#if visibleColumns.status}
                  <td class="p-4">
                    {#if row.status === 'Done'}
                      <span class="inline-flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-2.5 py-0.5 text-xs font-normal text-emerald-700 dark:text-emerald-400">
                        <svg class="size-3 text-emerald-600 dark:text-emerald-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
                          <polyline points="22 4 12 14.01 9 11.01"></polyline>
                        </svg>
                        Done
                      </span>
                    {:else}
                      <span class="inline-flex items-center gap-1.5 rounded-full border border-border bg-background px-2.5 py-0.5 text-xs font-normal text-muted-foreground">
                        <svg class="size-3 animate-spin text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                          <line x1="12" y1="2" x2="12" y2="6"></line>
                          <line x1="12" y1="18" x2="12" y2="22"></line>
                          <line x1="4.93" y1="4.93" x2="7.76" y2="7.76"></line>
                          <line x1="16.24" y1="16.24" x2="19.07" y2="19.07"></line>
                          <line x1="2" y1="12" x2="6" y2="12"></line>
                          <line x1="18" y1="12" x2="22" y2="12"></line>
                        </svg>
                        In Process
                      </span>
                    {/if}
                  </td>
                {/if}

                <!-- Target -->
                {#if visibleColumns.target}
                  <td class="p-4 text-right">
                    <input
                      type="text"
                      bind:value={row.target}
                      class="h-7 w-14 rounded border border-transparent bg-transparent px-1.5 text-right text-xs font-mono hover:border-input focus:border-input focus:bg-background focus:outline-none"
                    />
                  </td>
                {/if}

                <!-- Limit -->
                {#if visibleColumns.limit}
                  <td class="p-4 text-right">
                    <input
                      type="text"
                      bind:value={row.limit}
                      class="h-7 w-14 rounded border border-transparent bg-transparent px-1.5 text-right text-xs font-mono hover:border-input focus:border-input focus:bg-background focus:outline-none"
                    />
                  </td>
                {/if}

                <!-- Reviewer -->
                {#if visibleColumns.reviewer}
                  <td class="p-4 text-sm">
                    {#if row.reviewer === 'Assign reviewer'}
                      <select
                        bind:value={row.reviewer}
                        class="h-7 rounded border border-input bg-background px-2 text-xs text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      >
                        <option value="Assign reviewer">Assign reviewer</option>
                        <option value="Eddie Lake">Eddie Lake</option>
                        <option value="Jamik Tashpulatov">Jamik Tashpulatov</option>
                        <option value="Sarah Chen">Sarah Chen</option>
                      </select>
                    {:else}
                      <div class="flex items-center gap-2">
                        <div class="size-6 rounded-full bg-muted border border-border flex items-center justify-center text-[10px] font-medium text-foreground uppercase">
                          {row.reviewer[0]}
                        </div>
                        <span class="text-xs font-medium text-foreground">{row.reviewer}</span>
                      </div>
                    {/if}
                  </td>
                {/if}

                <!-- Actions Menu -->
                <td class="p-2 text-center">
                  <button
                    type="button"
                    class="size-7 inline-flex items-center justify-center rounded-md hover:bg-muted text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    title="Row options"
                    aria-label="Row options"
                  >
                    <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <circle cx="12" cy="12" r="1"></circle>
                      <circle cx="12" cy="5" r="1"></circle>
                      <circle cx="12" cy="19" r="1"></circle>
                    </svg>
                  </button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>

    <!-- DataTable Pagination Footer (Exact shadcn specification) -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-4 py-3 border-t border-border bg-card">
      <div class="text-xs text-muted-foreground">
        {selectedIds.length} of {filteredItems.length} row(s) selected.
      </div>

      <div class="flex items-center gap-6 text-xs text-foreground">
        <!-- Rows per page selector -->
        <div class="flex items-center gap-2">
          <span class="text-muted-foreground">Rows per page</span>
          <select
            bind:value={pageSize}
            onchange={() => { pageIndex = 0; }}
            class="h-8 rounded-md border border-input bg-background px-2 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value={10}>10</option>
            <option value={20}>20</option>
            <option value={50}>50</option>
          </select>
        </div>

        <!-- Page counter -->
        <div class="font-medium">
          Page {pageIndex + 1} of {totalPages}
        </div>

        <!-- Pagination Arrow Buttons -->
        <div class="flex items-center gap-1">
          <!-- First Page -->
          <button
            type="button"
            class="size-8 inline-flex items-center justify-center rounded-md border border-input bg-background text-foreground hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            disabled={pageIndex === 0}
            onclick={() => { pageIndex = 0; }}
            aria-label="First page"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="11 17 6 12 11 7"></polyline>
              <polyline points="18 17 13 12 18 7"></polyline>
            </svg>
          </button>

          <!-- Prev Page -->
          <button
            type="button"
            class="size-8 inline-flex items-center justify-center rounded-md border border-input bg-background text-foreground hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            disabled={pageIndex === 0}
            onclick={() => { pageIndex--; }}
            aria-label="Previous page"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
          </button>

          <!-- Next Page -->
          <button
            type="button"
            class="size-8 inline-flex items-center justify-center rounded-md border border-input bg-background text-foreground hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            disabled={pageIndex >= totalPages - 1}
            onclick={() => { pageIndex++; }}
            aria-label="Next page"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </button>

          <!-- Last Page -->
          <button
            type="button"
            class="size-8 inline-flex items-center justify-center rounded-md border border-input bg-background text-foreground hover:bg-accent disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer"
            disabled={pageIndex >= totalPages - 1}
            onclick={() => { pageIndex = totalPages - 1; }}
            aria-label="Last page"
          >
            <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="13 17 18 12 13 7"></polyline>
              <polyline points="6 17 11 12 6 7"></polyline>
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</div>

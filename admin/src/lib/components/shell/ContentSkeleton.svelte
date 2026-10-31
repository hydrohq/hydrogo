<script lang="ts">
  export type SkeletonKind = 'console' | 'dashboard' | 'table' | 'editor' | 'media' | 'settings' | 'auth';

  interface Props {
    kind?: SkeletonKind;
  }

  let { kind = 'dashboard' }: Props = $props();
</script>

{#if kind === 'console'}
  <!-- Console (Mağaza Fabrikası) Content Skeleton -->
  <div class="flex flex-1 flex-col min-w-0">
    <div class="@container/main flex flex-1 flex-col gap-6 p-4 md:p-6 lg:p-8 max-w-7xl mx-auto w-full min-w-0">
      <!-- Header Bar Skeleton -->
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border pb-5">
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-xl font-semibold tracking-tight text-foreground">Mağaza Fabrikası</h2>
            <div class="h-5 w-24 bg-muted/60 rounded-md border border-border animate-pulse"></div>
          </div>
          <p class="text-xs text-muted-foreground mt-1">
            Multi-tenant mağaza yönetimi ve alan adı altyapısı
          </p>
        </div>
        <div class="flex items-center gap-2.5">
          <div class="h-9 w-24 rounded-md border border-border bg-muted/30 animate-pulse"></div>
          <div class="h-9 w-36 rounded-md bg-foreground/15 animate-pulse"></div>
        </div>
      </div>

      <!-- 4 Console Metric Cards Skeleton -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <!-- Card 1: Aktif Mağaza -->
        <div class="p-4 rounded-lg border border-border bg-card shadow-2xs space-y-2">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>Aktif Mağaza</span>
            <span class="size-2 rounded-full bg-emerald-500/60 animate-ping"></span>
          </div>
          <div class="h-8 w-16 bg-muted/60 rounded animate-pulse"></div>
          <div class="h-3 w-32 bg-muted/40 rounded"></div>
        </div>

        <!-- Card 2: Kalan Kredi -->
        <div class="p-4 rounded-lg border border-border bg-card shadow-2xs space-y-2">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>Kalan Kredi</span>
            <div class="h-4 w-12 bg-muted/40 rounded"></div>
          </div>
          <div class="h-8 w-20 bg-muted/60 rounded animate-pulse"></div>
          <div class="h-3 w-28 bg-muted/40 rounded"></div>
        </div>

        <!-- Card 3: RAM / Sistem -->
        <div class="p-4 rounded-lg border border-border bg-card shadow-2xs space-y-2">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>RAM / Sistem</span>
            <div class="h-4 w-14 bg-muted/40 rounded"></div>
          </div>
          <div class="h-8 w-24 bg-muted/60 rounded animate-pulse"></div>
          <div class="h-3 w-36 bg-muted/40 rounded"></div>
        </div>

        <!-- Card 4: DNS / Wildcard -->
        <div class="p-4 rounded-lg border border-border bg-card shadow-2xs space-y-2">
          <div class="flex items-center justify-between text-xs text-muted-foreground">
            <span>DNS / SSL</span>
            <span class="size-2 rounded-full bg-emerald-500/60"></span>
          </div>
          <div class="h-8 w-28 bg-muted/60 rounded animate-pulse"></div>
          <div class="h-3 w-40 bg-muted/40 rounded"></div>
        </div>
      </div>

      <!-- Store Management Table Skeleton -->
      <div class="rounded-lg border border-border bg-card shadow-2xs overflow-hidden">
        <div class="p-3.5 border-b border-border flex flex-col sm:flex-row sm:items-center justify-between gap-3 bg-muted/10">
          <div class="h-8 w-64 sm:w-80 rounded-md bg-muted/40 border border-border"></div>
          <div class="flex items-center gap-1">
            {#each ['Tümü', 'Önizleme', 'Aktif', 'Askıda'] as tab, i}
              <div class="px-2.5 py-1 text-xs rounded-md {i === 0 ? 'bg-muted text-foreground font-medium' : 'text-muted-foreground'}">
                {tab}
              </div>
            {/each}
          </div>
        </div>

        <div class="overflow-x-auto">
          <table class="w-full text-xs text-left">
            <thead class="text-muted-foreground bg-muted/20 border-b border-border font-medium">
              <tr>
                <th class="px-4 py-2.5">Mağaza</th>
                <th class="px-4 py-2.5">Adres</th>
                <th class="px-4 py-2.5">Özel alan adı</th>
                <th class="px-4 py-2.5">Durum</th>
                <th class="px-4 py-2.5 text-center">Sipariş</th>
                <th class="px-4 py-2.5 text-right">İşlemler</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each [1, 2, 3] as _}
                <tr class="hover:bg-muted/10 transition-colors">
                  <td class="px-4 py-3">
                    <div class="h-3.5 w-32 bg-muted/60 rounded animate-pulse"></div>
                    <div class="h-2.5 w-24 bg-muted/30 rounded mt-1.5"></div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="h-3 w-40 bg-muted/40 rounded"></div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="h-3 w-28 bg-muted/30 rounded"></div>
                  </td>
                  <td class="px-4 py-3">
                    <div class="h-4 w-14 bg-muted/40 rounded-full"></div>
                  </td>
                  <td class="px-4 py-3 text-center">
                    <div class="h-3 w-6 bg-muted/40 rounded mx-auto"></div>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <div class="flex items-center justify-end gap-1">
                      <div class="h-6 w-14 bg-muted/40 rounded"></div>
                      <div class="h-6 w-14 bg-muted/40 rounded"></div>
                      <div class="h-6 w-14 bg-muted/40 rounded"></div>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>

{:else if kind === 'table'}
  <!-- Generic Collection / List View Skeleton -->
  <div class="p-4 md:p-6 lg:p-8 max-w-7xl mx-auto w-full space-y-5">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-border pb-4">
      <div class="space-y-1.5">
        <div class="h-6 w-36 bg-muted/70 rounded-md animate-pulse"></div>
        <div class="h-3 w-56 bg-muted/30 rounded"></div>
      </div>
      <div class="h-9 w-28 bg-primary/20 rounded-md animate-pulse"></div>
    </div>

    <!-- Toolbar -->
    <div class="flex items-center justify-between gap-3">
      <div class="h-8 w-64 bg-muted/40 rounded-md border border-border"></div>
      <div class="flex items-center gap-2">
        <div class="h-8 w-20 bg-muted/30 rounded-md border border-border"></div>
        <div class="h-8 w-24 bg-muted/30 rounded-md border border-border"></div>
      </div>
    </div>

    <!-- Data Table -->
    <div class="rounded-lg border border-border bg-card shadow-2xs overflow-hidden">
      <div class="h-10 bg-muted/20 border-b border-border flex items-center px-4 gap-4">
        <div class="size-4 bg-muted/40 rounded"></div>
        <div class="h-3 w-32 bg-muted/40 rounded"></div>
        <div class="h-3 w-24 bg-muted/40 rounded"></div>
        <div class="h-3 w-20 bg-muted/40 rounded ml-auto"></div>
      </div>
      <div class="divide-y divide-border">
        {#each [1, 2, 3, 4, 5] as _}
          <div class="p-3.5 px-4 flex items-center justify-between gap-4">
            <div class="flex items-center gap-3 flex-1">
              <div class="size-4 bg-muted/30 rounded"></div>
              <div class="space-y-1.5 flex-1">
                <div class="h-3.5 w-48 bg-muted/60 rounded animate-pulse"></div>
                <div class="h-2.5 w-28 bg-muted/30 rounded"></div>
              </div>
            </div>
            <div class="h-4 w-16 bg-muted/40 rounded-full"></div>
            <div class="h-3 w-24 bg-muted/30 rounded"></div>
            <div class="size-7 bg-muted/30 rounded"></div>
          </div>
        {/each}
      </div>
    </div>
  </div>

{:else if kind === 'editor'}
  <!-- Editor View Skeleton (Pages / News / Generic Item Editor) -->
  <div class="p-4 md:p-6 lg:p-8 max-w-5xl mx-auto w-full space-y-6">
    <div class="flex items-center justify-between border-b border-border pb-4">
      <div class="flex items-center gap-3">
        <div class="size-8 rounded-md bg-muted/40 border border-border"></div>
        <div class="h-6 w-40 bg-muted/70 rounded animate-pulse"></div>
      </div>
      <div class="flex items-center gap-2">
        <div class="h-8 w-20 rounded-md border border-border bg-muted/30"></div>
        <div class="h-8 w-24 rounded-md bg-primary/20 animate-pulse"></div>
      </div>
    </div>

    <!-- Title Input Skeleton -->
    <div class="space-y-2">
      <div class="h-10 w-full rounded-md border border-border bg-muted/20 animate-pulse"></div>
      <div class="h-3 w-48 bg-muted/30 rounded"></div>
    </div>

    <!-- TipTap Editor Toolbar + Body Skeleton -->
    <div class="rounded-lg border border-border bg-card shadow-2xs overflow-hidden">
      <div class="h-10 bg-muted/30 border-b border-border flex items-center gap-1.5 px-3">
        {#each [1, 2, 3, 4, 5, 6, 7] as _}
          <div class="size-7 rounded bg-muted/40"></div>
        {/each}
      </div>
      <div class="p-6 space-y-4 min-h-[300px]">
        <div class="h-4 w-5/6 bg-muted/40 rounded"></div>
        <div class="h-4 w-4/6 bg-muted/30 rounded"></div>
        <div class="h-4 w-full bg-muted/35 rounded"></div>
        <div class="h-4 w-3/5 bg-muted/25 rounded"></div>
      </div>
    </div>
  </div>

{:else if kind === 'media'}
  <!-- Media Library Grid Skeleton -->
  <div class="p-4 md:p-6 lg:p-8 max-w-7xl mx-auto w-full space-y-5">
    <div class="flex items-center justify-between border-b border-border pb-4">
      <div class="h-6 w-32 bg-muted/70 rounded animate-pulse"></div>
      <div class="h-8 w-28 bg-primary/20 rounded animate-pulse"></div>
    </div>
    <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
      {#each [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12] as _}
        <div class="rounded-lg border border-border bg-card p-2 space-y-2 shadow-2xs">
          <div class="aspect-square w-full rounded-md bg-muted/40 animate-pulse"></div>
          <div class="h-2.5 w-3/4 bg-muted/50 rounded"></div>
          <div class="h-2 w-1/2 bg-muted/30 rounded"></div>
        </div>
      {/each}
    </div>
  </div>

{:else if kind === 'settings'}
  <!-- Settings / Account Skeleton -->
  <div class="p-4 md:p-6 lg:p-8 max-w-4xl mx-auto w-full space-y-6">
    <div class="space-y-1.5 border-b border-border pb-4">
      <div class="h-6 w-36 bg-muted/70 rounded animate-pulse"></div>
      <div class="h-3 w-56 bg-muted/30 rounded"></div>
    </div>
    <div class="flex gap-2 border-b border-border pb-2">
      {#each ['Genel', 'Alan Adı', 'E-posta', 'Gelişmiş'] as tab, i}
        <div class="px-3 py-1.5 rounded-md text-xs {i === 0 ? 'bg-muted text-foreground' : 'text-muted-foreground'}">
          {tab}
        </div>
      {/each}
    </div>
    <div class="rounded-lg border border-border bg-card p-6 space-y-4 shadow-2xs">
      <div class="space-y-1.5">
        <div class="h-3 w-24 bg-muted/60 rounded"></div>
        <div class="h-9 w-full bg-muted/30 rounded-md border border-border"></div>
      </div>
      <div class="space-y-1.5">
        <div class="h-3 w-32 bg-muted/60 rounded"></div>
        <div class="h-24 w-full bg-muted/20 rounded-md border border-border"></div>
      </div>
    </div>
  </div>

{:else if kind === 'auth'}
  <!-- Auth / Login View Minimal Skeleton -->
  <div class="min-h-screen w-full flex flex-col items-center justify-center p-6 bg-muted/40">
    <div class="w-full max-w-sm flex flex-col gap-6">
      <div class="self-center h-8 w-28 bg-muted/50 rounded animate-pulse"></div>
      <div class="rounded-xl border border-border bg-card shadow-xs p-6 sm:p-8 space-y-4">
        <div class="space-y-2 text-center">
          <div class="h-6 w-32 bg-muted/70 rounded mx-auto animate-pulse"></div>
          <div class="h-3 w-48 bg-muted/30 rounded mx-auto"></div>
        </div>
        <div class="space-y-3 pt-2">
          <div class="h-9 w-full bg-muted/30 rounded-md border border-border"></div>
          <div class="h-9 w-full bg-muted/30 rounded-md border border-border"></div>
          <div class="h-9 w-full bg-primary/30 rounded-md animate-pulse"></div>
        </div>
      </div>
    </div>
  </div>

{:else}
  <!-- Default Dashboard Content Skeleton -->
  <div class="flex flex-1 flex-col">
    <div class="@container/main flex flex-1 flex-col gap-2">
      <div class="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
        <!-- 4 Section Metric Cards -->
        <div class="grid grid-cols-1 gap-4 px-4 @xl/main:grid-cols-2 @5xl/main:grid-cols-4 lg:px-6">
          {#each ['Sayfalar', 'Haberler', 'Üyeler', 'İstekler'] as title}
            <div class="rounded-xl border border-border bg-card p-6 shadow-xs relative">
              <div class="flex flex-col space-y-1.5">
                <span class="text-sm font-normal text-muted-foreground">{title}</span>
                <div class="h-8 w-16 bg-muted/60 rounded-md animate-pulse"></div>
              </div>
              <div class="flex flex-col items-start gap-1 text-sm pt-4">
                <div class="h-3 w-24 bg-muted/30 rounded"></div>
              </div>
            </div>
          {/each}
        </div>

        <!-- Chart Skeleton -->
        <div class="mx-4 lg:mx-6 rounded-xl border border-border bg-card shadow-xs p-6 space-y-4">
          <div class="flex items-center justify-between border-b border-border pb-3">
            <div class="space-y-1">
              <div class="h-4 w-32 bg-muted/70 rounded"></div>
              <div class="h-2.5 w-48 bg-muted/30 rounded"></div>
            </div>
            <div class="h-6 w-24 bg-muted/30 rounded"></div>
          </div>
          <div class="h-44 w-full flex flex-col justify-between py-2 relative overflow-hidden">
            <div class="border-b border-dashed border-border/50 w-full"></div>
            <div class="border-b border-dashed border-border/50 w-full"></div>
            <div class="border-b border-border w-full"></div>
          </div>
        </div>

        <!-- Recent Table Skeleton -->
        <div class="mx-4 lg:mx-6 rounded-xl border border-border bg-card shadow-xs overflow-hidden">
          <div class="h-10 bg-muted/20 border-b border-border flex items-center px-4 justify-between">
            <div class="h-4 w-28 bg-muted/50 rounded"></div>
            <div class="h-4 w-16 bg-muted/30 rounded"></div>
          </div>
          <div class="divide-y divide-border">
            {#each [1, 2, 3] as _}
              <div class="p-3.5 px-4 flex items-center justify-between">
                <div class="h-3.5 w-44 bg-muted/60 rounded"></div>
                <div class="h-3 w-16 bg-muted/30 rounded"></div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

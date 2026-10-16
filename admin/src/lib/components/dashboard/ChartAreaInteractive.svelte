<script lang="ts">
  let timeRange = $state<'90d' | '30d' | '7d'>('30d');
  let hoveredPoint = $state<{ x: number; y: number; date: string; desktop: number; mobile: number } | null>(null);

  // 90 days dataset matching official shadcn dashboard-01 chartData
  const rawData = [
    { date: "2024-04-01", desktop: 222, mobile: 150 },
    { date: "2024-04-05", desktop: 373, mobile: 290 },
    { date: "2024-04-10", desktop: 261, mobile: 190 },
    { date: "2024-04-15", desktop: 120, mobile: 170 },
    { date: "2024-04-20", desktop: 89, mobile: 150 },
    { date: "2024-04-25", desktop: 215, mobile: 250 },
    { date: "2024-04-30", desktop: 454, mobile: 380 },
    { date: "2024-05-05", desktop: 481, mobile: 390 },
    { date: "2024-05-10", desktop: 293, mobile: 330 },
    { date: "2024-05-15", desktop: 473, mobile: 380 },
    { date: "2024-05-20", desktop: 177, mobile: 230 },
    { date: "2024-05-25", desktop: 201, mobile: 250 },
    { date: "2024-05-30", desktop: 340, mobile: 280 },
    { date: "2024-06-05", desktop: 88, mobile: 140 },
    { date: "2024-06-10", desktop: 155, mobile: 200 },
    { date: "2024-06-15", desktop: 307, mobile: 350 },
    { date: "2024-06-20", desktop: 408, mobile: 450 },
    { date: "2024-06-25", desktop: 141, mobile: 190 },
    { date: "2024-06-30", desktop: 446, mobile: 400 },
  ];

  const filteredData = $derived.by(() => {
    if (timeRange === '7d') return rawData.slice(-6);
    if (timeRange === '30d') return rawData.slice(-11);
    return rawData;
  });

  const width = 800;
  const height = 240;
  const padding = { top: 20, right: 20, bottom: 30, left: 20 };

  const maxVal = $derived(Math.max(...filteredData.map(d => d.desktop + d.mobile), 600));

  function getX(i: number, total: number) {
    return padding.left + (i / (total - 1)) * (width - padding.left - padding.right);
  }

  function getY(val: number) {
    return height - padding.bottom - (val / maxVal) * (height - padding.top - padding.bottom);
  }

  const desktopPoints = $derived(
    filteredData.map((d, i) => ({
      x: getX(i, filteredData.length),
      y: getY(d.desktop + d.mobile),
      date: d.date,
      desktop: d.desktop,
      mobile: d.mobile,
    }))
  );

  const mobilePoints = $derived(
    filteredData.map((d, i) => ({
      x: getX(i, filteredData.length),
      y: getY(d.mobile),
    }))
  );

  function createAreaPath(points: { x: number; y: number }[], baselineY: number): string {
    if (points.length === 0) return '';
    const first = points[0];
    let path = `M ${first.x} ${first.y}`;
    for (let i = 1; i < points.length; i++) {
      const p = points[i];
      const prev = points[i - 1];
      const cx = (prev.x + p.x) / 2;
      path += ` C ${cx} ${prev.y}, ${cx} ${p.y}, ${p.x} ${p.y}`;
    }
    path += ` L ${points[points.length - 1].x} ${baselineY} L ${first.x} ${baselineY} Z`;
    return path;
  }

  function createLinePath(points: { x: number; y: number }[]): string {
    if (points.length === 0) return '';
    const first = points[0];
    let path = `M ${first.x} ${first.y}`;
    for (let i = 1; i < points.length; i++) {
      const p = points[i];
      const prev = points[i - 1];
      const cx = (prev.x + p.x) / 2;
      path += ` C ${cx} ${prev.y}, ${cx} ${p.y}, ${p.x} ${p.y}`;
    }
    return path;
  }

  const baseline = height - padding.bottom;
  const desktopArea = $derived(createAreaPath(desktopPoints, baseline));
  const desktopLine = $derived(createLinePath(desktopPoints));
  const mobileArea = $derived(createAreaPath(mobilePoints, baseline));
  const mobileLine = $derived(createLinePath(mobilePoints));
</script>

<div class="px-4 lg:px-6">
  <div class="rounded-xl border border-border bg-card text-card-foreground shadow-xs @container/card">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-6 border-b border-border/50">
      <div>
        <h3 class="text-base font-semibold leading-none tracking-tight text-foreground">Total Visitors</h3>
        <p class="text-sm text-muted-foreground mt-1.5">
          {#if timeRange === '90d'}Total for the last 3 months
          {:else if timeRange === '30d'}Total for the last 30 days
          {:else}Total for the last 7 days{/if}
        </p>
      </div>

      <!-- Toggle Group (Exact shadcn ToggleGroup variant="outline") -->
      <div class="flex items-center rounded-lg border border-border p-0.5 bg-background text-xs">
        <button
          type="button"
          class="px-3 py-1.5 rounded-md font-medium transition-colors {timeRange === '90d' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
          onclick={() => { timeRange = '90d'; }}
        >
          Last 3 months
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-md font-medium transition-colors {timeRange === '30d' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
          onclick={() => { timeRange = '30d'; }}
        >
          Last 30 days
        </button>
        <button
          type="button"
          class="px-3 py-1.5 rounded-md font-medium transition-colors {timeRange === '7d' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
          onclick={() => { timeRange = '7d'; }}
        >
          Last 7 days
        </button>
      </div>
    </div>

    <!-- Chart Container -->
    <div class="p-6 pt-4 relative">
      <div class="h-[250px] w-full relative">
        <svg viewBox="0 0 {width} {height}" class="w-full h-full overflow-visible" preserveAspectRatio="none">
          <defs>
            <linearGradient id="fillDesktop" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stop-color="hsl(var(--foreground))" stop-opacity="0.2"></stop>
              <stop offset="95%" stop-color="hsl(var(--foreground))" stop-opacity="0.0"></stop>
            </linearGradient>
            <linearGradient id="fillMobile" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stop-color="hsl(var(--muted-foreground))" stop-opacity="0.15"></stop>
              <stop offset="95%" stop-color="hsl(var(--muted-foreground))" stop-opacity="0.0"></stop>
            </linearGradient>
          </defs>

          <!-- Horizontal Grid Lines -->
          {#each [0.25, 0.5, 0.75, 1.0] as ratio}
            {@const y = height - padding.bottom - ratio * (height - padding.top - padding.bottom)}
            <line
              x1={padding.left}
              y1={y}
              x2={width - padding.right}
              y2={y}
              stroke="hsl(var(--border))"
              stroke-dasharray="3 3"
              stroke-width="1"
            />
          {/each}

          <!-- Mobile Area & Line -->
          <path d={mobileArea} fill="url(#fillMobile)"></path>
          <path d={mobileLine} fill="none" stroke="hsl(var(--muted-foreground))" stroke-width="2"></path>

          <!-- Desktop Area & Line -->
          <path d={desktopArea} fill="url(#fillDesktop)"></path>
          <path d={desktopLine} fill="none" stroke="hsl(var(--foreground))" stroke-width="2"></path>

          <!-- Interactive Hover Target Points -->
          {#each desktopPoints as pt}
            <circle
              cx={pt.x}
              cy={pt.y}
              r="4"
              role="presentation"
              class="fill-background stroke-foreground stroke-2 opacity-0 hover:opacity-100 transition-opacity cursor-pointer"
              onmouseenter={() => { hoveredPoint = pt; }}
              onmouseleave={() => { hoveredPoint = null; }}
            />
          {/each}
        </svg>

        <!-- Tooltip -->
        {#if hoveredPoint}
          <div
            class="absolute pointer-events-none rounded-lg border border-border bg-popover px-3 py-2 text-xs shadow-md z-20 text-popover-foreground transition-all"
            style="left: calc({(hoveredPoint.x / width) * 100}% - 60px); top: {hoveredPoint.y - 70}px;"
          >
            <div class="font-medium">{hoveredPoint.date}</div>
            <div class="flex items-center gap-2 mt-1">
              <span class="size-2 rounded-full bg-foreground"></span>
              <span class="text-muted-foreground">Desktop:</span>
              <span class="font-mono font-medium">{hoveredPoint.desktop}</span>
            </div>
            <div class="flex items-center gap-2 mt-0.5">
              <span class="size-2 rounded-full bg-muted-foreground"></span>
              <span class="text-muted-foreground">Mobile:</span>
              <span class="font-mono font-medium">{hoveredPoint.mobile}</span>
            </div>
          </div>
        {/if}
      </div>

      <!-- X-Axis Labels -->
      <div class="flex justify-between px-2 pt-2 text-[11px] font-mono text-muted-foreground">
        {#each filteredData.filter((_, i) => i % Math.ceil(filteredData.length / 6) === 0) as item}
          <span>{item.date.slice(5)}</span>
        {/each}
      </div>
    </div>
  </div>
</div>

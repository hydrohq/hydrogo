<script lang="ts">
  interface AnalyticsDailyRow {
    Day?: string;
    Metric: string;
    Dimension?: string;
    EventCount: number;
  }

  interface Props {
    analytics?: {
      unique_visitors_today?: number;
      daily?: AnalyticsDailyRow[];
    };
    runtime?: {
      started_at?: string;
      uptime_seconds?: number;
      requests?: number;
      views?: number;
      errors?: number;
      avg_latency_ms?: number;
    };
  }

  let { analytics, runtime }: Props = $props();

  let timeRange = $state<'90d' | '30d' | '7d'>('30d');
  let hoveredPoint = $state<{ x: number; y: number; date: string; desktop: number; mobile: number; total: number } | null>(null);

  function formatDateISO(d: Date): string {
    const year = d.getUTCFullYear();
    const month = String(d.getUTCMonth() + 1).padStart(2, '0');
    const day = String(d.getUTCDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  }

  function getDaysRange(count: number): string[] {
    const dates: string[] = [];
    const now = new Date();
    for (let i = count - 1; i >= 0; i--) {
      const d = new Date(now.getTime() - i * 86400000);
      dates.push(formatDateISO(d));
    }
    return dates;
  }

  const dayCount = $derived(timeRange === '7d' ? 7 : timeRange === '30d' ? 30 : 90);
  const dateSlots = $derived(getDaysRange(dayCount));
  const todayStr = $derived(formatDateISO(new Date()));

  // Process live analytics data from server (no mock data)
  const chartData = $derived.by(() => {
    const dailyRows = analytics?.daily || [];

    const desktopMap = new Map<string, number>();
    const mobileMap = new Map<string, number>();
    const viewsMap = new Map<string, number>();

    for (const row of dailyRows) {
      if (!row.Day) continue;
      if (row.Metric === 'device') {
        if (row.Dimension === 'desktop') {
          desktopMap.set(row.Day, (desktopMap.get(row.Day) || 0) + Number(row.EventCount));
        } else if (row.Dimension === 'mobile' || row.Dimension === 'tablet') {
          mobileMap.set(row.Day, (mobileMap.get(row.Day) || 0) + Number(row.EventCount));
        }
      } else if (row.Metric === 'daily_views') {
        viewsMap.set(row.Day, (viewsMap.get(row.Day) || 0) + Number(row.EventCount));
      }
    }

    return dateSlots.map((date) => {
      let desktop = desktopMap.get(date) || 0;
      let mobile = mobileMap.get(date) || 0;
      const views = viewsMap.get(date) || 0;

      // If views exist but device wasn't split, attribute to desktop
      if (views > 0 && desktop === 0 && mobile === 0) {
        desktop = views;
      }

      // If today has recorded unique visitors or runtime views, reflect them
      if (date === todayStr && desktop === 0 && mobile === 0) {
        if (analytics?.unique_visitors_today && analytics.unique_visitors_today > 0) {
          desktop = Number(analytics.unique_visitors_today);
        } else if (runtime?.views && runtime.views > 0) {
          desktop = Number(runtime.views);
        }
      }

      const total = desktop + mobile;
      return {
        date,
        desktop,
        mobile,
        total,
      };
    });
  });

  const totalVisitors = $derived(chartData.reduce((acc, d) => acc + d.total, 0));
  const totalDesktop = $derived(chartData.reduce((acc, d) => acc + d.desktop, 0));
  const totalMobile = $derived(chartData.reduce((acc, d) => acc + d.mobile, 0));

  const width = 800;
  const height = 240;
  const padding = { top: 20, right: 20, bottom: 30, left: 20 };

  const maxVal = $derived(Math.max(...chartData.map((d) => d.total), 10));

  function getX(i: number, total: number) {
    if (total <= 1) return padding.left;
    return padding.left + (i / (total - 1)) * (width - padding.left - padding.right);
  }

  function getY(val: number) {
    return height - padding.bottom - (val / maxVal) * (height - padding.top - padding.bottom);
  }

  const baseline = height - padding.bottom;

  const desktopPoints = $derived(
    chartData.map((d, i) => ({
      x: getX(i, chartData.length),
      y: getY(d.desktop + d.mobile),
      date: d.date,
      desktop: d.desktop,
      mobile: d.mobile,
      total: d.total,
    }))
  );

  const mobilePoints = $derived(
    chartData.map((d, i) => ({
      x: getX(i, chartData.length),
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

  const desktopArea = $derived(createAreaPath(desktopPoints, baseline));
  const desktopLine = $derived(createLinePath(desktopPoints));
  const mobileArea = $derived(createAreaPath(mobilePoints, baseline));
  const mobileLine = $derived(createLinePath(mobilePoints));

  function formatAxisDate(iso: string): string {
    const parts = iso.split('-');
    if (parts.length < 3) return iso;
    const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
    const m = parseInt(parts[1], 10) - 1;
    return `${months[m] || parts[1]} ${parseInt(parts[2], 10)}`;
  }

  const axisLabels = $derived.by(() => {
    const total = chartData.length;
    const count = total <= 7 ? total : 6;
    const step = Math.max(1, Math.floor((total - 1) / (count - 1)));
    const labels: { date: string; label: string; x: number }[] = [];
    for (let i = 0; i < total; i += step) {
      labels.push({
        date: chartData[i].date,
        label: formatAxisDate(chartData[i].date),
        x: getX(i, total),
      });
    }
    if (labels[labels.length - 1]?.date !== chartData[total - 1].date) {
      labels.push({
        date: chartData[total - 1].date,
        label: formatAxisDate(chartData[total - 1].date),
        x: getX(total - 1, total),
      });
    }
    return labels;
  });
</script>

<div class="px-4 lg:px-6">
  <div class="rounded-xl border border-border bg-card text-card-foreground shadow-xs @container/card">
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-6 border-b border-border/50">
      <div>
        <div class="flex items-center gap-2">
          <h3 class="text-base font-semibold leading-none tracking-tight text-foreground">Total Visitors</h3>
          <span class="inline-flex items-center gap-1 rounded-md bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-600 dark:text-emerald-400">
            <span class="size-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            Live Telemetry
          </span>
        </div>
        <p class="text-sm text-muted-foreground mt-1.5">
          {#if timeRange === '90d'}Total for the last 3 months ({totalVisitors.toLocaleString()} recorded visits)
          {:else if timeRange === '30d'}Total for the last 30 days ({totalVisitors.toLocaleString()} recorded visits)
          {:else}Total for the last 7 days ({totalVisitors.toLocaleString()} recorded visits){/if}
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-3">
        <!-- Live device breakdown pills if any traffic exists -->
        {#if totalVisitors > 0}
          <div class="hidden md:flex items-center gap-3 text-xs font-mono mr-2">
            <span class="flex items-center gap-1.5 text-foreground font-medium">
              <span class="size-2 rounded-full bg-foreground"></span>
              Desktop: {totalDesktop.toLocaleString()}
            </span>
            <span class="flex items-center gap-1.5 text-muted-foreground font-medium">
              <span class="size-2 rounded-full bg-muted-foreground"></span>
              Mobile: {totalMobile.toLocaleString()}
            </span>
          </div>
        {/if}

        <!-- Toggle Group (Exact shadcn ToggleGroup variant="outline") -->
        <div class="flex items-center rounded-lg border border-border p-0.5 bg-background text-xs">
          <button
            type="button"
            class="px-3 py-1.5 rounded-md font-medium transition-colors cursor-pointer {timeRange === '90d' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { timeRange = '90d'; }}
          >
            Last 3 months
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded-md font-medium transition-colors cursor-pointer {timeRange === '30d' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { timeRange = '30d'; }}
          >
            Last 30 days
          </button>
          <button
            type="button"
            class="px-3 py-1.5 rounded-md font-medium transition-colors cursor-pointer {timeRange === '7d' ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
            onclick={() => { timeRange = '7d'; }}
          >
            Last 7 days
          </button>
        </div>
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
            style="left: calc({(hoveredPoint.x / width) * 100}% - 60px); top: {Math.max(10, hoveredPoint.y - 70)}px;"
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
            <div class="flex items-center gap-2 mt-0.5 pt-1 border-t border-border/50">
              <span class="text-muted-foreground">Total:</span>
              <span class="font-mono font-medium">{hoveredPoint.total}</span>
            </div>
          </div>
        {/if}
      </div>

      <!-- X-Axis Labels positioned along the SVG coordinate system -->
      <div class="relative w-full h-6 mt-1 text-[11px] font-mono text-muted-foreground">
        {#each axisLabels as item}
          <span
            class="absolute -translate-x-1/2 whitespace-nowrap"
            style="left: {(item.x / width) * 100}%;"
          >
            {item.label}
          </span>
        {/each}
      </div>

      <!-- Clean footer note if 0 traffic recorded yet -->
      {#if totalVisitors === 0}
        <div class="mt-2 pt-3 border-t border-border/40 flex items-center justify-between text-xs text-muted-foreground">
          <div class="flex items-center gap-2">
            <span class="size-1.5 rounded-full bg-muted-foreground/50"></span>
            <span>0 visits recorded in this period. Public pages and REST API visits are captured in real time without cookies.</span>
          </div>
          <span class="font-mono text-[11px]">Today: {todayStr}</span>
        </div>
      {/if}
    </div>
  </div>
</div>

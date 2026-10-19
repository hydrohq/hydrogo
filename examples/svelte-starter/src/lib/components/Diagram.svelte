<script lang="ts">
  let { name }: { name: 'architecture' | 'publish-flow' } = $props();
</script>

<figure class="my-6 overflow-x-auto rounded-lg border border-line bg-subtle p-3 sm:p-5">
  {#if name === 'architecture'}
    <svg viewBox="0 0 780 400" class="mx-auto h-auto w-full min-w-[560px] max-w-[780px]" role="img" aria-labelledby="arch-t arch-d" font-family="ui-sans-serif, system-ui, sans-serif" font-size="13">
      <title id="arch-t">HydroGo architecture</title>
      <desc id="arch-d">Editors use the embedded admin panel and frontends read the REST API. Both are served by one Go binary that stores data in SQLite, PostgreSQL or MySQL and media on disk or S3.</desc>
      <defs>
        <marker id="arr" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M0 0 L10 5 L0 10 z" style="fill: var(--fg-faint)" />
        </marker>
      </defs>

      <!-- clients -->
      <g>
        <rect x="10" y="40" width="170" height="64" rx="10" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="95" y="68" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Editors</text>
        <text x="95" y="87" text-anchor="middle" style="fill: var(--fg-muted)">browser at /admin</text>

        <rect x="10" y="150" width="170" height="64" rx="10" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="95" y="178" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Your frontend</text>
        <text x="95" y="197" text-anchor="middle" style="fill: var(--fg-muted)">SvelteKit, Next, mobile…</text>

        <rect x="10" y="260" width="170" height="64" rx="10" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="95" y="288" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Scripts and CI</text>
        <text x="95" y="307" text-anchor="middle" style="fill: var(--fg-muted)">API keys, webhooks</text>
      </g>

      <!-- arrows in -->
      <g style="stroke: var(--fg-faint)" stroke-width="1.5" fill="none" marker-end="url(#arr)">
        <path d="M180 72 H260" />
        <path d="M180 182 H260" />
        <path d="M180 292 H260" />
      </g>

      <!-- binary -->
      <rect x="262" y="14" width="330" height="372" rx="14" style="fill: var(--accent-soft); stroke: var(--accent)" stroke-width="1.5" />
      <text x="427" y="38" text-anchor="middle" style="fill: var(--accent); font-weight: 700; letter-spacing: 0.04em">hydro (one Go binary)</text>

      <g>
        <rect x="282" y="54" width="140" height="48" rx="8" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="352" y="75" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Admin SPA</text>
        <text x="352" y="92" text-anchor="middle" style="fill: var(--fg-muted)">Svelte 5, /admin</text>

        <rect x="432" y="54" width="140" height="48" rx="8" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="502" y="75" text-anchor="middle" style="fill: var(--fg); font-weight: 600">REST API</text>
        <text x="502" y="92" text-anchor="middle" style="fill: var(--fg-muted)">/api/v1, JSON only</text>

        <rect x="282" y="120" width="290" height="40" rx="8" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="427" y="145" text-anchor="middle" style="fill: var(--fg)">Auth · CSRF · rate limits · audit trail</text>

        <rect x="282" y="176" width="290" height="100" rx="8" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="427" y="198" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Content modules</text>
        <text x="427" y="222" text-anchor="middle" style="fill: var(--fg-muted)">pages · news · members · site</text>
        <text x="427" y="244" text-anchor="middle" style="fill: var(--fg-muted)">media · content types (YAML)</text>
        <text x="427" y="264" text-anchor="middle" style="fill: var(--fg-muted)">search · jobs · webhooks</text>

        <rect x="282" y="292" width="290" height="80" rx="8" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="427" y="314" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Migrations and repositories</text>
        <text x="427" y="338" text-anchor="middle" style="fill: var(--fg-muted)">one SQL set per dialect</text>
        <text x="427" y="356" text-anchor="middle" style="fill: var(--fg-muted)">sqlite · postgres · mysql</text>
      </g>

      <!-- arrows out -->
      <g style="stroke: var(--fg-faint)" stroke-width="1.5" fill="none" marker-end="url(#arr)">
        <path d="M592 100 H660" />
        <path d="M592 190 H660" />
        <path d="M592 300 H660" />
      </g>

      <!-- storage -->
      <g>
        <rect x="662" y="60" width="108" height="80" rx="10" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="716" y="92" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Database</text>
        <text x="716" y="112" text-anchor="middle" style="fill: var(--fg-muted)">SQLite (default)</text>
        <text x="716" y="128" text-anchor="middle" style="fill: var(--fg-muted)">PostgreSQL · MySQL</text>

        <rect x="662" y="160" width="108" height="64" rx="10" style="fill: var(--bg-elevated); stroke: var(--border-strong)" />
        <text x="716" y="188" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Media</text>
        <text x="716" y="208" text-anchor="middle" style="fill: var(--fg-muted)">disk or S3</text>

        <rect x="662" y="262" width="108" height="64" rx="10" style="fill: var(--bg-elevated); stroke: var(--border-strong); stroke-dasharray: 5 4" />
        <text x="716" y="290" text-anchor="middle" style="fill: var(--fg); font-weight: 600">Redis</text>
        <text x="716" y="310" text-anchor="middle" style="fill: var(--fg-muted)">optional</text>
      </g>
    </svg>
    <figcaption class="mt-3 text-center text-xs text-faint">Everything inside the highlighted box ships as one executable.</figcaption>
  {:else}
    <svg viewBox="0 0 780 170" class="mx-auto h-auto w-full min-w-[560px] max-w-[780px]" role="img" aria-labelledby="flow-t flow-d" font-family="ui-sans-serif, system-ui, sans-serif" font-size="13">
      <title id="flow-t">Publish flow</title>
      <desc id="flow-d">An editor clicks Publish, HydroGo writes to the database, the next API request returns the new content, and an optional webhook can purge a cache.</desc>
      <defs>
        <marker id="arr2" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
          <path d="M0 0 L10 5 L0 10 z" style="fill: var(--fg-faint)" />
        </marker>
      </defs>
      {#each [{ x: 10, a: '1. Editor clicks Publish', b: 'admin panel' }, { x: 205, a: '2. Row is updated', b: 'database, one transaction' }, { x: 400, a: '3. Next API request', b: 'returns the new content' }, { x: 595, a: '4. Optional webhook', b: 'purge a cache, notify' }] as s, i}
        <rect x={s.x} y="45" width="175" height="70" rx="10" style="fill: var(--bg-elevated); stroke: {i === 3 ? 'var(--border-strong)' : 'var(--accent)'}; {i === 3 ? 'stroke-dasharray: 5 4' : ''}" stroke-width="1.5" />
        <text x={s.x + 87} y="76" text-anchor="middle" style="fill: var(--fg); font-weight: 600">{s.a}</text>
        <text x={s.x + 87} y="97" text-anchor="middle" style="fill: var(--fg-muted)">{s.b}</text>
        {#if i < 3}
          <path d="M{s.x + 177} 80 H{s.x + 203}" style="stroke: var(--fg-faint)" stroke-width="1.5" fill="none" marker-end="url(#arr2)" />
        {/if}
      {/each}
      <text x="390" y="150" text-anchor="middle" style="fill: var(--fg-faint)">No build, no deploy, no static regeneration in between.</text>
    </svg>
  {/if}
</figure>

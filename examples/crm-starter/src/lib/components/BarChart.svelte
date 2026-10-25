<script lang="ts">
  // One series, one hue; the tallest bar can be hatched to mark it.
  let { labels, values, highlight = -1, unit = '' }: { labels: string[]; values: number[]; highlight?: number; unit?: string } = $props();
  let W = $state(560);
  const H = $derived(W < 460 ? 170 : 190);
  const L = 30, B = 26, T = 8;
  const max = $derived(Math.ceil(Math.max(4, ...values) / 4) * 4);
  const ticks = $derived([0, 1, 2, 3, 4].map((i) => (max / 4) * i));
  const bw = $derived((W - L) / Math.max(1, labels.length));
  const y = (v: number) => T + (H - B - T) * (1 - v / max);
  let hover = $state(-1);
</script>

<div class="wrap" bind:clientWidth={W}>
  <svg viewBox="0 0 {W} {H}" width={W} height={H} role="img" aria-label={labels.map((l, i) => `${l} ${values[i]}`).join(', ')}>
    <defs>
      <pattern id="hatch" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
        <rect width="6" height="6" fill="var(--warm)" /><line x1="0" y1="0" x2="0" y2="6" stroke="#ffffff66" stroke-width="2" />
      </pattern>
    </defs>
    {#each ticks as t (t)}
      <line x1={L} x2={W} y1={y(t)} y2={y(t)} stroke="var(--line)" />
      <text x={L - 8} y={y(t) + 4} text-anchor="end" class="ax">{t}</text>
    {/each}
    {#each values as v, i (i)}
      {@const x = L + i * bw + bw * 0.2}
      {@const h = Math.max(v ? 3 : 0, (H - B - T) * (v / max))}
      <rect class="bar" style="--i:{i}" {x} y={H - B - h} width={bw * 0.6} height={h} rx={W < 460 ? 6 : 8} fill={i === highlight ? 'url(#hatch)' : 'var(--bar)'} opacity={hover >= 0 && hover !== i ? 0.55 : 1} />
      <rect x={L + i * bw} y={T} width={bw} height={H - B - T} fill="transparent" role="presentation" onmouseenter={() => (hover = i)} onmouseleave={() => (hover = -1)} />
      <text x={x + bw * 0.3} y={H - 8} text-anchor="middle" class="ax">{bw < 72 && labels[i].length > 5 ? `${labels[i].slice(0, 4)}.` : labels[i]}</text>
    {/each}
    {#if hover >= 0}
      {@const tx = Math.min(Math.max(L + hover * bw + bw / 2, 60), W - 60)}
      <g transform="translate({tx},{Math.max(14, y(values[hover]) - 34)})" pointer-events="none">
        <rect x="-56" y="-14" width="112" height="34" rx="9" fill="var(--tip)" />
        <text x="0" y="-0.5" text-anchor="middle" class="tip1">{labels[hover]}</text>
        <text x="0" y="13" text-anchor="middle" class="tip2">{values[hover]} {unit}</text>
      </g>
    {/if}
  </svg>
</div>

<style>
  .wrap { width: 100%; }
  svg { display: block; max-width: 100%; --bar: color-mix(in srgb, var(--accent) 22%, var(--surface)); --tip: #12313f; }
  .bar { transform-box: fill-box; transform-origin: bottom; animation: grow .7s var(--ease) both; animation-delay: calc(var(--i) * 55ms); transition: opacity .18s; }
  @keyframes grow { from { transform: scaleY(0); } to { transform: scaleY(1); } }
  .ax { fill: var(--muted); font-size: 11px; }
  @media (max-width: 480px) { .ax { font-size: 12px; } }
  .tip1 { fill: #fff; font-size: 11px; font-weight: 600; } .tip2 { fill: #c8d3d9; font-size: 11px; }
</style>

<script lang="ts">
  export interface Slice { label: string; value: number; color: string }
  let { slices, centerLabel = '' }: { slices: Slice[]; centerLabel?: string } = $props();
  const total = $derived(slices.reduce((a, s) => a + s.value, 0));
  const R = 70, r = 46, C = 90;
  let hover = $state(-1);

  function arc(from: number, to: number): string {
    const p = (a: number, rad: number) => [C + rad * Math.sin(a), C - rad * Math.cos(a)];
    const [x1, y1] = p(from, R), [x2, y2] = p(to, R), [x3, y3] = p(to, r), [x4, y4] = p(from, r);
    const big = to - from > Math.PI ? 1 : 0;
    return `M${x1} ${y1}A${R} ${R} 0 ${big} 1 ${x2} ${y2}L${x3} ${y3}A${r} ${r} 0 ${big} 0 ${x4} ${y4}Z`;
  }
  const arcs = $derived.by(() => {
    let acc = 0;
    const gap = slices.filter((s) => s.value > 0).length > 1 ? 0.035 : 0;
    return slices.map((s) => {
      const span = total ? (s.value / total) * Math.PI * 2 : 0;
      const a = { d: span > 0 ? arc(acc + gap / 2, acc + span - gap / 2) : '', s };
      acc += span;
      return a;
    });
  });
</script>

<div class="donut">
  <svg viewBox="0 0 180 180" role="img" aria-label={slices.map((s) => `${s.label} ${s.value}`).join(', ')}>
    {#if total === 0}<circle cx={C} cy={C} r={(R + r) / 2} fill="none" stroke="var(--line)" stroke-width={R - r} />{/if}
    {#each arcs as a, i (a.s.label)}
      {#if a.d}<path class="seg" style="--i:{i}" d={a.d} fill={a.s.color} opacity={hover >= 0 && hover !== i ? 0.45 : 1} role="presentation" onmouseenter={() => (hover = i)} onmouseleave={() => (hover = -1)} />{/if}
    {/each}
    <text x={C} y={C - 2} text-anchor="middle" class="big">{hover >= 0 ? slices[hover].value : total}</text>
    <text x={C} y={C + 15} text-anchor="middle" class="sub">{hover >= 0 ? slices[hover].label : centerLabel}</text>
  </svg>
  <ul>
    {#each slices as s, i (s.label)}
      <li onmouseenter={() => (hover = i)} onmouseleave={() => (hover = -1)}><span class="dot" style="background:{s.color}"></span>{s.label}<b>{s.value}</b></li>
    {/each}
  </ul>
</div>

<style>
  .donut { display: flex; align-items: center; gap: 1rem; flex-wrap: wrap; justify-content: center; }
  svg { width: 170px; height: 170px; flex: none; }
  .seg { transform-origin: 90px 90px; animation: seg .8s var(--ease) both; animation-delay: calc(var(--i) * 70ms); transition: opacity .18s; }
  @keyframes seg { from { opacity: 0; transform: rotate(-35deg) scale(.9); } to { opacity: 1; transform: none; } }
  .big { font-size: 26px; font-weight: 700; fill: var(--ink); } .sub { font-size: 10px; fill: var(--muted); }
  ul { list-style: none; margin: 0; padding: 0; display: grid; gap: .35rem; font-size: .85rem; min-width: 150px; }
  li { display: flex; align-items: center; gap: .5rem; } li b { margin-left: auto; padding-left: 1rem; }
  .dot { width: 10px; height: 10px; border-radius: 50%; }
  @media (max-width: 480px) { svg { width: 150px; height: 150px; } ul { min-width: 0; flex: 1; } }
</style>

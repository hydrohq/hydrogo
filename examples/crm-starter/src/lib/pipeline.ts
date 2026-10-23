export const STAGES = ['lead', 'qualified', 'proposal', 'negotiation', 'won', 'lost'] as const;
export type Stage = (typeof STAGES)[number];
export const OPEN_STAGES: Stage[] = ['lead', 'qualified', 'proposal', 'negotiation'];

export interface DealLike {
  id: string;
  stage?: unknown;
  amount?: unknown;
  currency?: unknown;
}

/** Group deals by stage; unknown stages are dropped. */
export function byStage<T extends DealLike>(deals: T[]): Record<Stage, T[]> {
  const columns = Object.fromEntries(STAGES.map((stage) => [stage, [] as T[]])) as Record<Stage, T[]>;
  for (const deal of deals) {
    const stage = deal.stage as Stage;
    if (stage in columns) columns[stage].push(deal);
  }
  return columns;
}

/** Sum amounts per currency (amounts in different currencies are never added together). */
export function totals(deals: DealLike[]): Record<string, number> {
  const sums: Record<string, number> = {};
  for (const deal of deals) {
    const amount = typeof deal.amount === 'number' ? deal.amount : 0;
    if (!amount) continue;
    const currency = typeof deal.currency === 'string' && deal.currency ? deal.currency : 'TRY';
    sums[currency] = (sums[currency] ?? 0) + amount;
  }
  return sums;
}

export function formatMoney(sums: Record<string, number>, lang: string): string {
  const parts = Object.entries(sums).map(([currency, value]) =>
    // Whole amounts stay compact; cents are shown when there are any.
    new Intl.NumberFormat(lang, { style: 'currency', currency, minimumFractionDigits: Number.isInteger(value) ? 0 : 2, maximumFractionDigits: Number.isInteger(value) ? 0 : 2 }).format(value)
  );
  return parts.length ? parts.join(' · ') : '–';
}

export function isOverdue(due: unknown, now = new Date()): boolean {
  return typeof due === 'string' && !Number.isNaN(Date.parse(due)) && new Date(due) <= now;
}

/** Share of closed deals that were won; null with nothing closed yet to divide by. */
export function winRate(wonCount: number, lostCount: number): number | null {
  const closed = wonCount + lostCount;
  return closed > 0 ? wonCount / closed : null;
}

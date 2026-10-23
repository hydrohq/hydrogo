import { describe, expect, it } from 'vitest';
import { STAGES, byStage, formatMoney, isOverdue, totals } from './pipeline';

describe('pipeline', () => {
  it('groups deals by stage and drops unknown stages', () => {
    const columns = byStage([{ id: '1', stage: 'lead' }, { id: '2', stage: 'won' }, { id: '3', stage: 'weird' }]);
    expect(Object.keys(columns)).toEqual([...STAGES]);
    expect(columns.lead.map((d) => d.id)).toEqual(['1']);
    expect(columns.won).toHaveLength(1);
  });

  it('never adds amounts in different currencies together', () => {
    expect(totals([{ id: '1', amount: 100, currency: 'USD' }, { id: '2', amount: 50, currency: 'USD' }, { id: '3', amount: 10, currency: 'EUR' }, { id: '4' }])).toEqual({ USD: 150, EUR: 10 });
  });

  it('formats an empty total as a dash', () => {
    expect(formatMoney({}, 'en')).toBe('–');
    expect(formatMoney({ USD: 1500 }, 'en')).toContain('1,500');
    expect(formatMoney({ TRY: 1234.5 }, 'en')).toContain('1,234.50');
  });

  it('detects overdue activities', () => {
    const now = new Date('2026-01-02T00:00:00Z');
    expect(isOverdue('2026-01-01T00:00:00Z', now)).toBe(true);
    expect(isOverdue('2026-02-01T00:00:00Z', now)).toBe(false);
    expect(isOverdue(undefined, now)).toBe(false);
  });
});

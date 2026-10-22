import { describe, expect, it } from 'vitest';
import type { Field } from '$lib/api/types';
import { expandParam, isManyRelation, recordTitle, relationCell } from './relations';

const one: Field = { id: 'company', label: 'Company', kind: 'relation', relation: { collection: 'companies' } as never };
const many: Field = { id: 'tags', label: 'Tags', kind: 'relation', relation: { collection: 'tags', cardinality: 'many' } as never };
const titles = (collection: string) => ({ companies: 'name', tags: 'label' } as Record<string, string>)[collection];

describe('relations', () => {
  it('tells a single relation from a many relation', () => {
    expect(isManyRelation(one)).toBe(false);
    expect(isManyRelation(many)).toBe(true);
    expect(isManyRelation({ id: 'x', label: 'X', kind: 'text' })).toBe(false);
  });

  it('asks the server to expand only relation fields, at most five', () => {
    const fields: Field[] = [{ id: 'name', label: 'Name', kind: 'text' }, one, many];
    expect(expandParam(fields)).toBe('company,tags');
    const six = Array.from({ length: 6 }, (_, i) => ({ ...one, id: `r${i}` }));
    expect(expandParam(six).split(',')).toHaveLength(5);
    expect(expandParam([])).toBe('');
  });

  it('names a record by its title field, else its id', () => {
    expect(recordTitle({ id: 'r_1', name: 'Acme' }, 'name')).toBe('Acme');
    expect(recordTitle({ id: 'r_1', name: '  ' }, 'name')).toBe('r_1');
    expect(recordTitle({ id: 'r_1' }, undefined)).toBe('r_1');
    expect(recordTitle(undefined, 'name')).toBe('');
  });

  it('shows the expanded title in a list cell and falls back to the id', () => {
    expect(relationCell({ company: 'r_1', expanded: { company: { id: 'r_1', name: 'Acme' } } }, one, titles)).toBe('Acme');
    expect(relationCell({ company: 'r_1' }, one, titles)).toBe('r_1');
    expect(relationCell({ company: '' }, one, titles)).toBe('—');
    expect(relationCell({ tags: ['a', 'b'], expanded: { tags: [{ id: 'a', label: 'Hot' }, { id: 'b', label: 'Cold' }] } }, many, titles)).toBe('Hot, Cold');
    expect(relationCell({ tags: [] }, many, titles)).toBe('—');
  });
});

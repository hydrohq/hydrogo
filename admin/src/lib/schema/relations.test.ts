import { describe, expect, it } from 'vitest';
import type { Field } from '$lib/api/types';
import {
  expandParam,
  isManyRelation,
  isRelation,
  parseRelationKeys,
  recordTitle,
  relationCell,
  relationValueField,
  resolveRecordMedia,
  serializeRelationKeys,
} from './relations';

const one: Field = { id: 'company', label: 'Company', kind: 'relation', relation: { collection: 'companies' } as never };
const many: Field = { id: 'tags', label: 'Tags', kind: 'relation', relation: { collection: 'tags', cardinality: 'many' } as never };
const textSlugMany: Field = {
  id: 'related_products',
  label: 'Related',
  kind: 'text',
  widget: 'relation',
  relation: { collection: 'products', cardinality: 'many', value_field: 'slug' },
};
const titles = (collection: string) => ({ companies: 'name', tags: 'label', products: 'name' } as Record<string, string>)[collection];

describe('relations', () => {
  it('tells a single relation from a many relation', () => {
    expect(isManyRelation(one)).toBe(false);
    expect(isManyRelation(many)).toBe(true);
    expect(isRelation(textSlugMany)).toBe(true);
    expect(isManyRelation(textSlugMany)).toBe(true);
    expect(isManyRelation({ id: 'x', label: 'X', kind: 'text' })).toBe(false);
  });

  it('determines value_field for native and widget relations', () => {
    expect(relationValueField(one)).toBe('id');
    expect(relationValueField(textSlugMany)).toBe('slug');
    expect(
      relationValueField({
        id: 'product_slug',
        label: 'Product',
        kind: 'text',
        widget: 'relation',
        relation: { collection: 'products' },
      })
    ).toBe('slug');
  });

  it('parses and serializes relation keys for id and text-slug modes', () => {
    expect(parseRelationKeys(['a', 'b'], 'id', true)).toEqual(['a', 'b']);
    expect(parseRelationKeys('belt-a, belt-b, belt-a', 'slug', true)).toEqual(['belt-a', 'belt-b']);
    expect(parseRelationKeys('', 'slug', true)).toEqual([]);
    expect(parseRelationKeys('belt-a', 'slug', false)).toEqual(['belt-a']);

    expect(serializeRelationKeys(['a', 'b'], 'id', true)).toEqual(['a', 'b']);
    expect(serializeRelationKeys(['a'], 'id', false)).toBe('a');
    expect(serializeRelationKeys(['belt-a', 'belt-b'], 'slug', true)).toBe('belt-a, belt-b');
    expect(serializeRelationKeys([], 'slug', true)).toBe('');
  });

  it('asks the server to expand only native relation fields, at most five', () => {
    const fields: Field[] = [{ id: 'name', label: 'Name', kind: 'text' }, one, many, textSlugMany];
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

  it('resolves record thumbnail URLs safely', () => {
    expect(resolveRecordMedia({ cover_image: '/media/a.jpg' })).toBe('/media/a.jpg');
    expect(resolveRecordMedia({ image: 'asset_123' })).toBe('/media/asset_123');
    expect(resolveRecordMedia({})).toBe('');
  });

  it('shows the expanded title or lookup title in a list cell and falls back to raw value', () => {
    expect(relationCell({ company: 'r_1', expanded: { company: { id: 'r_1', name: 'Acme' } } }, one, titles)).toBe('Acme');
    expect(relationCell({ company: 'r_1' }, one, titles)).toBe('r_1');
    expect(relationCell({ company: '' }, one, titles)).toBe('—');
    expect(relationCell({ tags: ['a', 'b'], expanded: { tags: [{ id: 'a', label: 'Hot' }, { id: 'b', label: 'Cold' }] } }, many, titles)).toBe('Hot, Cold');
    expect(relationCell({ tags: [] }, many, titles)).toBe('—');
    expect(
      relationCell(
        { related_products: 'belt-a, belt-b' },
        textSlugMany,
        titles,
        (target, key) => (target === 'products' ? ({ 'belt-a': 'Kemer A' } as Record<string, string>)[key] : undefined)
      )
    ).toBe('Kemer A, belt-b');
  });
});

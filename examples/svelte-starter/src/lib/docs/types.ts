/**
 * Documentation content is authored as typed data. Inline markup inside text:
 *   `code`, **bold**, [label](https://external) and [label](doc:section/page#heading)
 */
export type CalloutKind = 'note' | 'tip' | 'warning';

export type CodeSample = { label: string; lang: string; code: string; title?: string };

export type Block =
  | { type: 'p'; text: string }
  | { type: 'h2' | 'h3'; text: string; id: string }
  | { type: 'code'; lang: string; code: string; title?: string }
  | { type: 'tabs'; tabs: CodeSample[] }
  | { type: 'callout'; kind: CalloutKind; title?: string; text: string }
  | { type: 'table'; head: string[]; rows: string[][] }
  | { type: 'list'; items: string[]; ordered: boolean }
  | { type: 'cards'; items: { title: string; text: string; href: string }[] }
  | { type: 'diagram'; name: 'architecture' | 'publish-flow' };

export interface DocPage {
  id: string;
  title: string;
  description: string;
  blocks: Block[];
}

export interface DocSection {
  id: string;
  title: string;
  pages: DocPage[];
}

export const slugify = (text: string): string =>
  text
    .toLowerCase()
    .replace(/`/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');

export const p = (text: string): Block => ({ type: 'p', text });
export const h2 = (text: string, id = slugify(text)): Block => ({ type: 'h2', text, id });
export const h3 = (text: string, id = slugify(text)): Block => ({ type: 'h3', text, id });
export const code = (lang: string, source: string, title?: string): Block => ({ type: 'code', lang, code: source.replace(/^\n/, '').replace(/\s+$/, ''), title });
export const tabs = (...items: CodeSample[]): Block => ({ type: 'tabs', tabs: items.map((t) => ({ ...t, code: t.code.replace(/^\n/, '').replace(/\s+$/, '') })) });
export const note = (text: string, title?: string): Block => ({ type: 'callout', kind: 'note', title, text });
export const tip = (text: string, title?: string): Block => ({ type: 'callout', kind: 'tip', title, text });
export const warn = (text: string, title?: string): Block => ({ type: 'callout', kind: 'warning', title, text });
export const table = (head: string[], ...rows: string[][]): Block => ({ type: 'table', head, rows });
export const ul = (...items: string[]): Block => ({ type: 'list', items, ordered: false });
export const ol = (...items: string[]): Block => ({ type: 'list', items, ordered: true });
export const cards = (...items: { title: string; text: string; href: string }[]): Block => ({ type: 'cards', items });
export const diagram = (name: 'architecture' | 'publish-flow'): Block => ({ type: 'diagram', name });
export const page = (id: string, title: string, description: string, ...blocks: Block[]): DocPage => ({ id, title, description, blocks });

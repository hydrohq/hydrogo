import type { Block, DocPage, DocSection } from './types';
import { plain } from './inline';
import { introduction } from './introduction';
import { gettingStarted } from './getting-started';
import { configuration } from './configuration';
import { contentTypes } from './content-types';
import { restApi } from './rest-api';
import { adminGuide } from './admin-guide';
import { frontends } from './frontends';
import { multiLanguage } from './multi-language';
import { security } from './security';
import { deployment } from './deployment';
import { upgrading } from './upgrading';
import { reference } from './reference';

export const sections: DocSection[] = [
  introduction,
  gettingStarted,
  configuration,
  contentTypes,
  restApi,
  adminGuide,
  frontends,
  multiLanguage,
  security,
  deployment,
  upgrading,
  reference,
];

export interface FlatPage {
  section: DocSection;
  page: DocPage;
  index: number;
}

export const flatPages: FlatPage[] = sections.flatMap((section) => section.pages.map((page) => ({ section, page, index: 0 }))).map((entry, index) => ({ ...entry, index }));

export const firstPage = flatPages[0];

export function findPage(sectionId: string, pageId: string): FlatPage | undefined {
  const inSection = sectionId ? sections.find((s) => s.id === sectionId) : undefined;
  if (!inSection) return undefined;
  const target = pageId || inSection.pages[0]?.id;
  return flatPages.find((f) => f.section.id === sectionId && f.page.id === target);
}

export function neighbours(entry: FlatPage): { prev?: FlatPage; next?: FlatPage } {
  return { prev: flatPages[entry.index - 1], next: flatPages[entry.index + 1] };
}

export function headingsOf(page: DocPage): { id: string; text: string; level: 2 | 3 }[] {
  return page.blocks.flatMap((b) => (b.type === 'h2' || b.type === 'h3' ? [{ id: b.id, text: b.text, level: (b.type === 'h2' ? 2 : 3) as 2 | 3 }] : []));
}

// ---------------------------------------------------------------- search

interface SearchEntry {
  section: DocSection;
  page: DocPage;
  anchor: string;
  heading: string;
  text: string;
  haystack: string;
}

export interface SearchHit {
  section: DocSection;
  page: DocPage;
  anchor: string;
  heading: string;
  snippet: string;
  score: number;
}

function blockText(block: Block): string {
  switch (block.type) {
    case 'p':
      return plain(block.text);
    case 'h2':
    case 'h3':
      return block.text;
    case 'code':
      return block.code;
    case 'tabs':
      return block.tabs.map((t) => t.code).join('\n');
    case 'callout':
      return plain(block.text);
    case 'table':
      return [...block.head, ...block.rows.flat().map(plain)].join(' ');
    case 'list':
      return block.items.map(plain).join(' ');
    case 'cards':
      return block.items.map((i) => `${i.title} ${i.text}`).join(' ');
    default:
      return '';
  }
}

let index: SearchEntry[] | null = null;

function buildIndex(): SearchEntry[] {
  const entries: SearchEntry[] = [];
  for (const { section, page } of flatPages) {
    let current: SearchEntry = { section, page, anchor: '', heading: '', text: page.description, haystack: '' };
    entries.push(current);
    for (const block of page.blocks) {
      if (block.type === 'h2' || block.type === 'h3') {
        current = { section, page, anchor: block.id, heading: block.text, text: '', haystack: '' };
        entries.push(current);
      } else {
        current.text += ` ${blockText(block)}`;
      }
    }
  }
  for (const e of entries) e.haystack = `${e.page.title} ${e.heading} ${e.text}`.toLowerCase();
  return entries;
}

function snippetFor(text: string, terms: string[]): string {
  const clean = text.replace(/\s+/g, ' ').trim();
  const lower = clean.toLowerCase();
  let at = -1;
  for (const t of terms) {
    const i = lower.indexOf(t);
    if (i !== -1 && (at === -1 || i < at)) at = i;
  }
  if (at === -1) return clean.slice(0, 120);
  const start = Math.max(0, at - 40);
  return (start > 0 ? '…' : '') + clean.slice(start, start + 140) + (start + 140 < clean.length ? '…' : '');
}

export function search(query: string, limit = 12): SearchHit[] {
  const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (terms.length === 0) return [];
  index ??= buildIndex();
  const hits: SearchHit[] = [];
  for (const e of index) {
    if (!terms.every((t) => e.haystack.includes(t))) continue;
    const title = e.page.title.toLowerCase();
    const heading = e.heading.toLowerCase();
    let score = 0;
    for (const t of terms) {
      if (title.includes(t)) score += 10;
      if (heading.includes(t)) score += 8;
      if (e.text.toLowerCase().includes(t)) score += 2;
    }
    if (!e.anchor) score += 3;
    if (title === query.toLowerCase().trim()) score += 20;
    hits.push({ section: e.section, page: e.page, anchor: e.anchor, heading: e.heading, snippet: snippetFor(e.text || e.page.description, terms), score });
  }
  hits.sort((a, b) => b.score - a.score);
  return hits.slice(0, limit);
}

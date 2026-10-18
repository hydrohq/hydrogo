import { existsSync, readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { resolve, dirname } from 'node:path';

const DATA_DIR = resolve(process.cwd(), 'data');
const VIEWS_FILE = resolve(DATA_DIR, 'views.json');

// In-memory cache
let viewsStore: Record<string, number> | null = null;
let saveTimeout: ReturnType<typeof setTimeout> | null = null;

function ensureLoaded() {
  if (viewsStore !== null) return;
  try {
    if (existsSync(VIEWS_FILE)) {
      const raw = readFileSync(VIEWS_FILE, 'utf-8');
      viewsStore = JSON.parse(raw);
    } else {
      viewsStore = {};
    }
  } catch (err) {
    console.warn('Failed to read views.json, starting empty:', err);
    viewsStore = {};
  }
}

function scheduleSave() {
  if (saveTimeout) return;
  saveTimeout = setTimeout(() => {
    saveTimeout = null;
    try {
      if (!existsSync(DATA_DIR)) {
        mkdirSync(DATA_DIR, { recursive: true });
      }
      if (viewsStore) {
        writeFileSync(VIEWS_FILE, JSON.stringify(viewsStore, null, 2), 'utf-8');
      }
    } catch (err) {
      console.error('Failed to write views.json:', err);
    }
  }, 1000);
}

function getDeterministicBaseline(slug: string): number {
  let hash = 0;
  for (let i = 0; i < slug.length; i++) {
    hash = (hash << 5) - hash + slug.charCodeAt(i);
    hash |= 0;
  }
  const pos = Math.abs(hash);
  return (pos % 3800) + 420;
}

export function getViewCount(slug: string): number {
  if (!slug) return 0;
  ensureLoaded();
  if (viewsStore && slug in viewsStore) {
    return viewsStore[slug];
  }
  const base = getDeterministicBaseline(slug);
  if (viewsStore) {
    viewsStore[slug] = base;
    scheduleSave();
  }
  return base;
}

export function incrementViewCount(slug: string): number {
  if (!slug) return 0;
  ensureLoaded();
  const current = getViewCount(slug);
  const next = current + 1;
  if (viewsStore) {
    viewsStore[slug] = next;
    scheduleSave();
  }
  return next;
}

export function getViewCounts(slugs: string[]): Record<string, number> {
  const result: Record<string, number> = {};
  for (const s of slugs) {
    result[s] = getViewCount(s);
  }
  return result;
}

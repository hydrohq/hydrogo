export type Segment =
  | { kind: 'text'; text: string }
  | { kind: 'code'; text: string }
  | { kind: 'bold'; text: string }
  | { kind: 'link'; text: string; href: string; external: boolean };

const TOKEN = /(`[^`]+`)|(\*\*[^*]+\*\*)|(\[[^\]]+\]\([^)\s]+\))/g;

/** doc:section/page#heading -> #/docs/section/page/heading */
export function resolveHref(raw: string): { href: string; external: boolean } {
  if (raw.startsWith('doc:')) {
    const [path, anchor] = raw.slice(4).split('#');
    return { href: `#/docs/${path}${anchor ? `/${anchor}` : ''}`, external: false };
  }
  if (raw.startsWith('/admin')) return { href: raw, external: true };
  return { href: raw, external: /^https?:/.test(raw) };
}

export function parseInline(text: string): Segment[] {
  const out: Segment[] = [];
  let last = 0;
  for (const m of text.matchAll(TOKEN)) {
    const index = m.index ?? 0;
    if (index > last) out.push({ kind: 'text', text: text.slice(last, index) });
    if (m[1]) out.push({ kind: 'code', text: m[1].slice(1, -1) });
    else if (m[2]) out.push({ kind: 'bold', text: m[2].slice(2, -2) });
    else if (m[3]) {
      const close = m[3].indexOf('](');
      const label = m[3].slice(1, close);
      const target = resolveHref(m[3].slice(close + 2, -1));
      out.push({ kind: 'link', text: label, ...target });
    }
    last = index + m[0].length;
  }
  if (last < text.length) out.push({ kind: 'text', text: text.slice(last) });
  return out;
}

/** Plain text of an inline string (used by search and aria labels). */
export function plain(text: string): string {
  return parseInline(text)
    .map((s) => s.text)
    .join('');
}

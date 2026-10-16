/**
 * Normalizes rich text content from various formats (legacy Quill Delta JSON,
 * plain text, or HTML) into clean semantic HTML ready for TipTap editor.
 */

export interface QuillOp {
  insert: string | { image?: string; [key: string]: any };
  attributes?: {
    header?: 1 | 2 | 3 | 4 | 5 | 6;
    bold?: boolean;
    italic?: boolean;
    underline?: boolean;
    strike?: boolean;
    code?: boolean;
    blockquote?: boolean;
    'code-block'?: boolean;
    list?: 'bullet' | 'ordered';
    link?: string;
    [key: string]: any;
  };
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function formatInline(text: string, attrs?: QuillOp['attributes']): string {
  if (!text) return '';
  let formatted = escapeHtml(text);
  if (!attrs) return formatted;
  if (attrs.code) formatted = `<code>${formatted}</code>`;
  if (attrs.bold) formatted = `<strong>${formatted}</strong>`;
  if (attrs.italic) formatted = `<em>${formatted}</em>`;
  if (attrs.underline) formatted = `<u>${formatted}</u>`;
  if (attrs.strike) formatted = `<s>${formatted}</s>`;
  if (attrs.link) formatted = `<a href="${escapeHtml(attrs.link)}" target="_blank" rel="noopener noreferrer">${formatted}</a>`;
  return formatted;
}

/**
 * Converts Quill Delta ops into semantic HTML.
 */
export function deltaToHtml(ops: QuillOp[]): string {
  if (!Array.isArray(ops) || ops.length === 0) return '';

  const lines: Array<{ inlines: Array<{ html: string }>; block: QuillOp['attributes'] }> = [];
  let currentInlines: Array<{ html: string }> = [];

  for (const op of ops) {
    if (typeof op.insert !== 'string') {
      if (op.insert && typeof op.insert === 'object' && op.insert.image) {
        currentInlines.push({ html: `<img src="${escapeHtml(String(op.insert.image))}" alt="" />` });
      }
      continue;
    }

    const str = op.insert;
    const parts = str.split('\n');

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      if (part.length > 0) {
        currentInlines.push({ html: formatInline(part, op.attributes) });
      }

      // Each newline signals the end of a line; block attributes come from op.attributes
      if (i < parts.length - 1) {
        lines.push({
          inlines: currentInlines,
          block: op.attributes || {},
        });
        currentInlines = [];
      }
    }
  }

  if (currentInlines.length > 0) {
    lines.push({ inlines: currentInlines, block: {} });
  }

  const htmlParts: string[] = [];
  let currentListType: 'ul' | 'ol' | null = null;

  for (const line of lines) {
    const content = line.inlines.map((i) => i.html).join('');
    const block = line.block || {};

    if (block.list) {
      const listTag = block.list === 'ordered' ? 'ol' : 'ul';
      if (currentListType !== listTag) {
        if (currentListType) htmlParts.push(`</${currentListType}>`);
        htmlParts.push(`<${listTag}>`);
        currentListType = listTag;
      }
      htmlParts.push(`<li>${content || '<br>'}</li>`);
      continue;
    } else if (currentListType) {
      htmlParts.push(`</${currentListType}>`);
      currentListType = null;
    }

    if (!content.trim() && !block.header && !block.blockquote && !block['code-block']) {
      continue;
    }

    if (block.header) {
      const level = Math.min(Math.max(block.header, 1), 3);
      htmlParts.push(`<h${level}>${content}</h${level}>`);
    } else if (block.blockquote) {
      htmlParts.push(`<blockquote><p>${content}</p></blockquote>`);
    } else if (block['code-block']) {
      htmlParts.push(`<pre><code>${content}</code></pre>`);
    } else {
      htmlParts.push(`<p>${content}</p>`);
    }
  }

  if (currentListType) {
    htmlParts.push(`</${currentListType}>`);
  }

  return htmlParts.join('');
}

/**
 * Normalizes input value for TipTap: detects legacy Quill Delta JSON or plain text
 * and outputs HTML.
 */
export function normalizeRichText(val: string | undefined | null): string {
  if (!val) return '';
  const trimmed = val.trim();
  if (!trimmed) return '';

  // 1. Detect Quill Delta JSON
  if (trimmed.startsWith('{') && (trimmed.includes('"ops"') || trimmed.includes('"document"'))) {
    try {
      const parsed = JSON.parse(trimmed);
      const ops = parsed.ops || parsed.document;
      if (Array.isArray(ops)) {
        return deltaToHtml(ops);
      }
    } catch {
      // not valid JSON, proceed
    }
  }

  // 2. Already HTML: contains tags like <p>, <h1>, <div>, <ul>, etc.
  if (/<[a-z][\s\S]*>/i.test(trimmed)) {
    return trimmed;
  }

  // 3. Plain text: wrap lines in <p>
  const paragraphs = trimmed
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter(Boolean);

  if (paragraphs.length > 0) {
    return paragraphs.map((p) => `<p>${escapeHtml(p).replace(/\n/g, '<br>')}</p>`).join('');
  }

  return `<p>${escapeHtml(trimmed)}</p>`;
}

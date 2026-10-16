import { describe, it, expect } from 'vitest';
import { normalizeRichText, deltaToHtml } from './deltaToHtml';

describe('deltaToHtml', () => {
  it('converts basic text with headers and paragraphs', () => {
    const ops = [
      { insert: 'Header Text\n', attributes: { header: 1 as const } },
      { insert: 'This is a normal paragraph.\n' },
    ];
    const html = deltaToHtml(ops);
    expect(html).toContain('<h1>Header Text</h1>');
    expect(html).toContain('<p>This is a normal paragraph.</p>');
  });

  it('converts inline styles like bold, italic, code, link', () => {
    const ops = [
      { insert: 'Normal ' },
      { insert: 'Bold', attributes: { bold: true } },
      { insert: ' and ' },
      { insert: 'Italic', attributes: { italic: true } },
      { insert: ' and ' },
      { insert: 'Code', attributes: { code: true } },
      { insert: ' and ' },
      { insert: 'Link', attributes: { link: 'https://example.com' } },
      { insert: '\n' },
    ];
    const html = deltaToHtml(ops);
    expect(html).toContain('<strong>Bold</strong>');
    expect(html).toContain('<em>Italic</em>');
    expect(html).toContain('<code>Code</code>');
    expect(html).toContain('<a href="https://example.com"');
  });

  it('converts bullet and ordered lists', () => {
    const ops = [
      { insert: 'Item 1\n', attributes: { list: 'bullet' as const } },
      { insert: 'Item 2\n', attributes: { list: 'bullet' as const } },
      { insert: 'Step 1\n', attributes: { list: 'ordered' as const } },
    ];
    const html = deltaToHtml(ops);
    expect(html).toContain('<ul><li>Item 1</li><li>Item 2</li></ul>');
    expect(html).toContain('<ol><li>Step 1</li></ol>');
  });

  it('converts blockquotes and code blocks', () => {
    const ops = [
      { insert: 'A wise quote\n', attributes: { blockquote: true } },
      { insert: 'const a = 1;\n', attributes: { 'code-block': true } },
    ];
    const html = deltaToHtml(ops);
    expect(html).toContain('<blockquote><p>A wise quote</p></blockquote>');
    expect(html).toContain('<pre><code>const a = 1;</code></pre>');
  });
});

describe('normalizeRichText', () => {
  it('returns empty string for empty input', () => {
    expect(normalizeRichText('')).toBe('');
    expect(normalizeRichText(null)).toBe('');
    expect(normalizeRichText(undefined)).toBe('');
  });

  it('parses legacy Quill delta JSON string from seed data', () => {
    const legacySeed = JSON.stringify({
      ops: [
        { insert: 'Güvenilir ve Esnek E-Ticaret Altyapısı\n', attributes: { header: 1 } },
        { insert: 'Commerce modülü modern e-ticaret operasyonlarının tüm ihtiyaçlarını karşılar.\n' },
      ],
    });
    const html = normalizeRichText(legacySeed);
    expect(html).toContain('<h1>Güvenilir ve Esnek E-Ticaret Altyapısı</h1>');
    expect(html).toContain('<p>Commerce modülü modern e-ticaret operasyonlarının tüm ihtiyaçlarını karşılar.</p>');
  });

  it('preserves existing HTML', () => {
    const html = '<p>Already <strong>HTML</strong></p>';
    expect(normalizeRichText(html)).toBe(html);
  });

  it('wraps plain text into paragraphs', () => {
    const text = 'First paragraph\n\nSecond paragraph';
    const html = normalizeRichText(text);
    expect(html).toBe('<p>First paragraph</p><p>Second paragraph</p>');
  });
});

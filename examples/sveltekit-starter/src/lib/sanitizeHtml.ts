/**
 * Defense in depth for rich text rendered with {@html}: HydroGo already
 * sanitizes rich-text fields on write and on read (internal/richtext), so
 * this should never have anything to strip. It exists for the case that
 * matters most -- a HydroGo version this starter talks to that predates
 * that fix, or a direct API write that bypassed it.
 */
export function sanitizeHtml(dirty?: string | null): string {
  if (!dirty) return '';
  return dirty
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    .replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '')
    .replace(/<\/?(iframe|object|embed|form|input|button|base|meta|link)\b[^>]*>/gi, '')
    .replace(/\son\w+\s*=\s*(?:'[^']*'|"[^"]*"|[^\s>]+)/gi, '')
    .replace(/(href|src)\s*=\s*(['"])\s*javascript:[^'"]*\2/gi, '$1="#"');
}

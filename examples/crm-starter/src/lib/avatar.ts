const PALETTE = ['#1b6a94', '#b9693a', '#2f8f5b', '#6250d6', '#c4477c', '#0e8a9c', '#8a6d1f', '#5b7f2a'];

/** Two letters from a name ("Jane Doe" -> "JD"), falling back to the first letters of one word. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

/** The same name always gets the same color (a simple hash into a fixed palette). */
export function colorFor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
  return PALETTE[hash % PALETTE.length];
}

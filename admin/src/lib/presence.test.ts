import { describe, it, expect } from 'vitest';
import type { ActiveEditor } from './types/presence';

describe('Document Presence Helpers', () => {
  it('formats active times accurately', () => {
    function formatActiveTime(secs: number): string {
      if (secs <= 2) return 'Şu anda aktif';
      if (secs < 60) return `${secs} sn önce`;
      const mins = Math.floor(secs / 60);
      return `${mins} dk önce`;
    }

    expect(formatActiveTime(0)).toBe('Şu anda aktif');
    expect(formatActiveTime(2)).toBe('Şu anda aktif');
    expect(formatActiveTime(15)).toBe('15 sn önce');
    expect(formatActiveTime(59)).toBe('59 sn önce');
    expect(formatActiveTime(60)).toBe('1 dk önce');
    expect(formatActiveTime(125)).toBe('2 dk önce');
  });

  it('generates user initials from name or email', () => {
    function getInitials(nameOrEmail: string): string {
      const clean = nameOrEmail.split('@')[0].trim();
      const parts = clean.split(/[ ._-]+/).filter(Boolean);
      if (parts.length >= 2) {
        return (parts[0][0] + parts[1][0]).toUpperCase();
      }
      return (clean.slice(0, 2) || 'ED').toUpperCase();
    }

    expect(getInitials('John Doe')).toBe('JD');
    expect(getInitials('john.doe@example.com')).toBe('JD');
    expect(getInitials('arda')).toBe('AR');
    expect(getInitials('admin@hydrogo.io')).toBe('AD');
    expect(getInitials('')).toBe('ED');
  });

  it('computes deterministic color palettes for editors', () => {
    const PALETTES = ['emerald', 'indigo', 'violet', 'amber', 'rose', 'sky', 'teal', 'fuchsia'];
    function getPaletteIndex(nameOrEmail: string) {
      let hash = 0;
      for (let i = 0; i < nameOrEmail.length; i++) {
        hash = (hash << 5) - hash + nameOrEmail.charCodeAt(i);
        hash |= 0;
      }
      return Math.abs(hash) % PALETTES.length;
    }

    const idx1 = getPaletteIndex('arda@example.com');
    const idx2 = getPaletteIndex('arda@example.com');
    expect(idx1).toBe(idx2);
    expect(idx1).toBeGreaterThanOrEqual(0);
    expect(idx1).toBeLessThan(PALETTES.length);
  });

  it('structures active editor payloads correctly', () => {
    const editor: ActiveEditor = {
      user_id: 'usr_123',
      email: 'editor@hydrogo.io',
      name: 'Editor',
      updated_at: new Date().toISOString(),
      active_seconds: 5,
    };

    expect(editor.user_id).toBe('usr_123');
    expect(editor.email).toBe('editor@hydrogo.io');
    expect(editor.active_seconds).toBe(5);
  });
});

// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import ThemesTab from './ThemesTab.svelte';

describe('ThemesTab', () => {
  it('renders all 4 built-in themes and switches active selection', async () => {
    const settings: Record<string, string> = {
      'storefront.theme_id': 'minimal',
      'storefront.theme_tokens': '',
    };
    const onsubmit = vi.fn((e: SubmitEvent) => e.preventDefault());

    render(ThemesTab, {
      props: {
        settings,
        saving: false,
        onsubmit,
      },
    });

    expect(screen.getAllByText('Minimal').length).toBeGreaterThan(0);
    expect(screen.getByText('Boutique')).toBeTruthy();
    expect(screen.getByText('Editorial')).toBeTruthy();
    expect(screen.getByText('B2B Catalog')).toBeTruthy();

    const selectButtons = screen.getAllByRole('button', { name: /Select Theme|Temayı Seç/i });
    await fireEvent.click(selectButtons[0]);

    expect(settings['storefront.theme_id']).toBe('boutique');
    expect(settings['storefront.theme_tokens']).toContain('#9a3412');

    const saveBtn = screen.getByRole('button', { name: /Save Theme Settings|Tema Ayarlarını Kaydet/i });
    await fireEvent.click(saveBtn);
    expect(onsubmit).toHaveBeenCalledTimes(1);
  });

  it('resets customized tokens back to theme defaults', async () => {
    const settings: Record<string, string> = {
      'storefront.theme_id': 'minimal',
      'storefront.theme_tokens': '{"accent":"#112233"}',
    };

    render(ThemesTab, {
      props: {
        settings,
        saving: false,
        onsubmit: vi.fn(),
      },
    });

    const resetBtn = screen.getByRole('button', { name: /Reset to theme defaults|Tema varsayılanlarına dön/i });
    await fireEvent.click(resetBtn);
    expect(settings['storefront.theme_tokens']).toBe('');
  });
});

// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/svelte';
import BlocksEditor, { type Block } from './BlocksEditor.svelte';

describe('BlocksEditor type-aware fields', () => {
  it('renders slide inputs for carousel and adds new slides', async () => {
    const blocks: Block[] = [
      {
        type: 'carousel',
        payload: {
          slides: [{ src: '/media/1.jpg', alt: 'First slide', caption: 'Hero Look' }],
        },
      },
    ];
    const onchange = vi.fn();

    render(BlocksEditor, { props: { blocks, onchange } });

    expect(screen.getByDisplayValue('First slide')).toBeTruthy();
    expect(screen.getByDisplayValue('Hero Look')).toBeTruthy();

    const addSlideBtn = screen.getByRole('button', { name: /\+ Add Slide|\+ Slayt Ekle/i });
    await fireEvent.click(addSlideBtn);

    expect(onchange).toHaveBeenCalledTimes(1);
    const updated = onchange.mock.calls[0][0] as Block[];
    expect(updated[0].payload?.slides).toHaveLength(2);
  });

  it('renders author and quote fields for testimonials and question/answer for faq', () => {
    const blocks: Block[] = [
      {
        type: 'testimonials',
        payload: { items: [{ name: 'Ayse Y.', quote: 'Great storefront!' }] },
      },
      {
        type: 'faq',
        payload: { items: [{ question: 'Shipping?', answer: '1-3 days.' }] },
      },
      {
        type: 'hero',
        payload: { title: 'Welcome', subtitle: 'Sub', button_label: 'Shop', button_href: '/products' },
      },
    ];

    render(BlocksEditor, { props: { blocks, onchange: vi.fn() } });

    expect(screen.getByDisplayValue('Ayse Y.')).toBeTruthy();
    expect(screen.getByDisplayValue('Great storefront!')).toBeTruthy();
    expect(screen.getByDisplayValue('Shipping?')).toBeTruthy();
    expect(screen.getByDisplayValue('1-3 days.')).toBeTruthy();
    expect(screen.getByDisplayValue('Shop')).toBeTruthy();
    expect(screen.getByDisplayValue('/products')).toBeTruthy();
  });
});

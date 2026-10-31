// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import AIAssistantModal from './AIAssistantModal.svelte';

vi.mock('$lib/api/client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
  },
}));

describe('AIAssistantModal', () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it('renders modal when open with operation buttons', () => {
    render(AIAssistantModal, {
      props: {
        open: true,
        pageId: 'page-1',
        currentTitle: 'Sample Title',
        currentBody: 'Sample Body Content',
        onapply: vi.fn(),
        onclose: vi.fn(),
      },
    });

    expect(screen.getByText('✨ AI Content Assistant')).toBeTruthy();
    expect(screen.getByText('Polish & Refine')).toBeTruthy();
    expect(screen.getByText('SEO Optimize')).toBeTruthy();
    expect(screen.getByText('Summarize')).toBeTruthy();
    expect(screen.getByText('Translate')).toBeTruthy();
    expect(screen.getByText('Generate Suggestion')).toBeTruthy();
  });

  it('generates suggestion and applies it to editor', async () => {
    const onapply = vi.fn();
    const onclose = vi.fn();

    (api.post as any).mockResolvedValueOnce({
      suggestion: {
        title: 'Refined Title',
        body: 'Refined Body Content',
      },
    });

    render(AIAssistantModal, {
      props: {
        open: true,
        pageId: 'page-1',
        currentTitle: 'Original Title',
        currentBody: 'Original Body',
        onapply,
        onclose,
      },
    });

    // Click Generate Suggestion
    const generateBtn = screen.getByText('Generate Suggestion');
    await fireEvent.click(generateBtn);

    await waitFor(() => {
      expect(screen.getByText('Refined Title')).toBeTruthy();
      expect(screen.getByText('Refined Body Content')).toBeTruthy();
    });

    // Click Apply to Editor
    const applyBtn = screen.getByText('✓ Apply to Editor');
    await fireEvent.click(applyBtn);

    expect(onapply).toHaveBeenCalledWith('Refined Title', 'Refined Body Content');
    expect(onclose).toHaveBeenCalled();
  });
});

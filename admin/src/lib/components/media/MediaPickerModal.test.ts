// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { api } from '$lib/api/client';
import MediaPickerModal from './MediaPickerModal.svelte';

vi.mock('$lib/api/client', () => ({
  api: { get: vi.fn(), upload: vi.fn() },
}));

function asset(over: Partial<any> = {}) {
  return { id: 'm1', original_filename: 'photo.png', mime_type: 'image/png', ...over };
}

describe('MediaPickerModal', () => {
  afterEach(() => {
    vi.mocked(api.get).mockReset();
    vi.mocked(api.upload).mockReset();
  });

  it('loads media once opened and shows the empty state when there is none', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [] });

    // Act
    render(MediaPickerModal, { props: { open: true, onselect: vi.fn(), onclose: vi.fn() } });

    // Assert
    await waitFor(() => expect(screen.getByText(/No media assets found/)).toBeTruthy());
  });

  it('renders loaded assets and filters them by the search box', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [asset(), asset({ id: 'm2', original_filename: 'report.pdf', mime_type: 'application/pdf' })] });
    render(MediaPickerModal, { props: { open: true, onselect: vi.fn(), onclose: vi.fn() } });
    await waitFor(() => expect(screen.getByText('photo.png')).toBeTruthy());

    // Act
    await fireEvent.input(screen.getByPlaceholderText('Filter assets...'), { target: { value: 'report' } });

    // Assert
    expect(screen.queryAllByText('photo.png')).toHaveLength(0);
    expect(screen.getAllByText('report.pdf').length).toBeGreaterThan(0);
  });

  it('selects an item and confirms it through onselect/onclose', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [asset()] });
    const onselect = vi.fn();
    const onclose = vi.fn();
    render(MediaPickerModal, { props: { open: true, onselect, onclose } });
    await waitFor(() => expect(screen.getByText('photo.png')).toBeTruthy());

    // Act
    await fireEvent.click(screen.getByText('photo.png').closest('button')!);
    await fireEvent.click(screen.getByText('Choose Asset'));

    // Assert
    expect(onselect).toHaveBeenCalledWith(expect.objectContaining({ id: 'm1' }));
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('disables "Choose Asset" until something is selected', async () => {
    // Arrange
    vi.mocked(api.get).mockResolvedValue({ data: [asset()] });
    render(MediaPickerModal, { props: { open: true, onselect: vi.fn(), onclose: vi.fn() } });
    await waitFor(() => expect(screen.getByText('photo.png')).toBeTruthy());

    // Assert
    expect((screen.getByText('Choose Asset') as HTMLElement).closest('button')).toHaveProperty('disabled', true);
  });
});

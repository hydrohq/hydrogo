// @vitest-environment jsdom
import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SlashCommandMenu, { type SlashItem } from './SlashCommandMenu.svelte';

function items(): SlashItem[] {
  return [
    { id: 'h1', title: 'Heading 1', description: 'Big heading', badge: 'H1', action: vi.fn() },
    { id: 'h2', title: 'Heading 2', description: 'Medium heading', badge: 'H2', action: vi.fn() },
  ];
}

describe('SlashCommandMenu', () => {
  it('renders every item with its title and description', () => {
    // Arrange
    const props = { items: items(), selectedIndex: 0, position: { top: 0, left: 0 }, onSelect: vi.fn(), onHover: vi.fn() };

    // Act
    render(SlashCommandMenu, { props });

    // Assert
    expect(screen.getByText('Heading 1')).toBeTruthy();
    expect(screen.getByText('Medium heading')).toBeTruthy();
  });

  it('positions itself using the given top/left offsets', () => {
    // Arrange
    const props = { items: items(), selectedIndex: 0, position: { top: 120, left: 40 }, onSelect: vi.fn(), onHover: vi.fn() };

    // Act
    const { container } = render(SlashCommandMenu, { props });

    // Assert
    const menu = container.firstElementChild as HTMLElement;
    expect(menu.style.top).toBe('120px');
    expect(menu.style.left).toBe('40px');
  });

  it('calls onSelect with the clicked item', async () => {
    // Arrange
    const onSelect = vi.fn();
    const list = items();
    const props = { items: list, selectedIndex: 0, position: { top: 0, left: 0 }, onSelect, onHover: vi.fn() };
    render(SlashCommandMenu, { props });

    // Act
    await fireEvent.click(screen.getByText('Heading 2'));

    // Assert
    expect(onSelect).toHaveBeenCalledWith(list[1]);
  });

  it('calls onHover with the index when the mouse enters an item', async () => {
    // Arrange
    const onHover = vi.fn();
    const props = { items: items(), selectedIndex: 0, position: { top: 0, left: 0 }, onSelect: vi.fn(), onHover };
    render(SlashCommandMenu, { props });

    // Act
    await fireEvent.mouseEnter(screen.getByText('Heading 2').closest('button')!);

    // Assert
    expect(onHover).toHaveBeenCalledWith(1);
  });
});

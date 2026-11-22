<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';
  import type { Editor } from '@tiptap/core';

  interface Props {
    editor: Editor;
    position: { top: number; left: number };
    onOpenLink: () => void;
  }

  let { editor, position, onOpenLink }: Props = $props();
</script>

<div
  class="absolute z-30 flex items-center gap-0.5 rounded-lg border border-border bg-popover/95 backdrop-blur-sm p-1 text-popover-foreground shadow-lg animate-in fade-in zoom-in-95 duration-100"
  style="top: {position.top}px; left: {position.left}px;"
  role="toolbar"
  aria-label={t('editor.floating')}
>
  <button
    type="button"
    class="size-7 inline-flex items-center justify-center rounded hover:bg-accent text-xs font-bold transition-colors {editor.isActive('bold') ? 'bg-muted text-primary' : ''}"
    onclick={() => editor.chain().focus().toggleBold().run()}
    title={t('editor.boldTip')}
    aria-label={t('editor.bold')}
  >
    B
  </button>
  <button
    type="button"
    class="size-7 inline-flex items-center justify-center rounded hover:bg-accent text-xs italic transition-colors {editor.isActive('italic') ? 'bg-muted text-primary' : ''}"
    onclick={() => editor.chain().focus().toggleItalic().run()}
    title={t('editor.italicTip')}
    aria-label={t('editor.italic')}
  >
    I
  </button>
  <button
    type="button"
    class="size-7 inline-flex items-center justify-center rounded hover:bg-accent text-xs line-through transition-colors {editor.isActive('strike') ? 'bg-muted text-primary' : ''}"
    onclick={() => editor.chain().focus().toggleStrike().run()}
    title={t('editor.strike')}
    aria-label={t('editor.strike')}
  >
    S
  </button>
  <button
    type="button"
    class="size-7 inline-flex items-center justify-center rounded hover:bg-accent text-xs font-mono transition-colors {editor.isActive('code') ? 'bg-muted text-primary' : ''}"
    onclick={() => editor.chain().focus().toggleCode().run()}
    title={t('editor.inlineCode')}
    aria-label={t('editor.inlineCode')}
  >
    &lt;&gt;
  </button>
  <button
    type="button"
    class="size-7 inline-flex items-center justify-center rounded hover:bg-accent text-xs transition-colors {editor.isActive('link') ? 'bg-muted text-primary' : ''}"
    onclick={onOpenLink}
    title={t('editor.insertLink')}
    aria-label={t('editor.insertLink')}
  >
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="size-3.5" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
  </button>

  <span class="w-px h-3.5 bg-border mx-0.5"></span>

  <button
    type="button"
    class="px-1.5 h-7 inline-flex items-center justify-center rounded hover:bg-accent text-[11px] font-semibold transition-colors {editor.isActive('heading', { level: 2 }) ? 'bg-muted text-primary' : ''}"
    onclick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
    title={t('editor.h2')}
  >
    H2
  </button>
  <button
    type="button"
    class="px-1.5 h-7 inline-flex items-center justify-center rounded hover:bg-accent text-[11px] font-semibold transition-colors {editor.isActive('blockquote') ? 'bg-muted text-primary' : ''}"
    onclick={() => editor.chain().focus().toggleBlockquote().run()}
    title={t('editor.quote')}
  >
    {t('editor.quote')}
  </button>
</div>

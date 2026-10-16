<script lang="ts">
  import type { Editor } from '@tiptap/core';

  interface Props {
    editor: Editor;
  }

  let { editor }: Props = $props();

  let showLinkInput = $state(false);
  let linkUrl = $state('');

  function openLinkInput() {
    linkUrl = editor.getAttributes('link').href || '';
    showLinkInput = true;
  }

  function applyLink() {
    if (linkUrl.trim() === '') {
      editor.chain().focus().extendMarkRange('link').unsetLink().run();
    } else {
      let finalUrl = linkUrl.trim();
      if (!/^https?:\/\//i.test(finalUrl) && !finalUrl.startsWith('/') && !finalUrl.startsWith('#')) {
        finalUrl = 'https://' + finalUrl;
      }
      editor.chain().focus().extendMarkRange('link').setLink({ href: finalUrl }).run();
    }
    showLinkInput = false;
    linkUrl = '';
  }

  function removeLink() {
    editor.chain().focus().unsetLink().run();
    showLinkInput = false;
  }
</script>

<div class="flex flex-col border-b border-border bg-muted/30 select-none">
  <div class="flex flex-wrap items-center gap-0.5 p-1.5 text-muted-foreground">
    <!-- Undo / Redo -->
    <button
      type="button"
      disabled={!editor.can().undo()}
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground disabled:opacity-30 disabled:pointer-events-none transition-colors cursor-pointer"
      onclick={() => editor.chain().focus().undo().run()}
      title="Undo (Ctrl+Z)"
      aria-label="Undo"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 7v6h6" />
        <path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13" />
      </svg>
    </button>
    <button
      type="button"
      disabled={!editor.can().redo()}
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground disabled:opacity-30 disabled:pointer-events-none transition-colors cursor-pointer"
      onclick={() => editor.chain().focus().redo().run()}
      title="Redo (Ctrl+Y)"
      aria-label="Redo"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 7v6h-6" />
        <path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3l3 2.7" />
      </svg>
    </button>

    <span class="w-[1px] h-4 bg-border mx-1"></span>

    <!-- Headings -->
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('paragraph') && !editor.isActive('heading') ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
      onclick={() => editor.chain().focus().setParagraph().run()}
      title="Paragraph (Normal Text)"
    >
      P
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('heading', { level: 1 }) ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
      onclick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
      title="Heading 1"
    >
      H1
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('heading', { level: 2 }) ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
      onclick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
      title="Heading 2"
    >
      H2
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('heading', { level: 3 }) ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
      onclick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
      title="Heading 3"
    >
      H3
    </button>

    <span class="w-[1px] h-4 bg-border mx-1"></span>

    <!-- Formatting: Bold, Italic, Strike, Code -->
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md font-bold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('bold') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleBold().run()}
      title="Bold (Ctrl+B)"
      aria-label="Bold"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="M6 12h9a4 4 0 0 1 0 8H6v-8Z" />
        <path d="M6 4h8a4 4 0 0 1 0 8H6V4Z" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md italic hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('italic') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleItalic().run()}
      title="Italic (Ctrl+I)"
      aria-label="Italic"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <line x1="19" y1="4" x2="10" y2="4" />
        <line x1="14" y1="20" x2="5" y2="20" />
        <line x1="15" y1="4" x2="9" y2="20" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('strike') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleStrike().run()}
      title="Strikethrough"
      aria-label="Strikethrough"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
        <path d="M16 4H9a3 3 0 0 0-2.83 4" />
        <path d="M14 12a4 4 0 0 1 0 8H6" />
        <line x1="4" y1="12" x2="20" y2="12" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md font-mono text-xs hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('code') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleCode().run()}
      title="Inline Code"
      aria-label="Inline Code"
    >
      &lt;/&gt;
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('link') ? 'bg-background text-primary shadow-xs' : ''}"
      onclick={openLinkInput}
      title="Link"
      aria-label="Insert Link"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
        <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
      </svg>
    </button>

    <span class="w-[1px] h-4 bg-border mx-1"></span>

    <!-- Blocks: Lists, Quote, Code, Divider -->
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('bulletList') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleBulletList().run()}
      title="Bullet List"
      aria-label="Bullet List"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="8" y1="6" x2="21" y2="6" />
        <line x1="8" y1="12" x2="21" y2="12" />
        <line x1="8" y1="18" x2="21" y2="18" />
        <line x1="3" y1="6" x2="3.01" y2="6" />
        <line x1="3" y1="12" x2="3.01" y2="12" />
        <line x1="3" y1="18" x2="3.01" y2="18" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('orderedList') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleOrderedList().run()}
      title="Numbered List"
      aria-label="Numbered List"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="10" y1="6" x2="21" y2="6" />
        <line x1="10" y1="12" x2="21" y2="12" />
        <line x1="10" y1="18" x2="21" y2="18" />
        <path d="M4 6h1v4" />
        <path d="M4 10h2" />
        <path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('blockquote') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleBlockquote().run()}
      title="Quote / Callout"
      aria-label="Quote"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z" />
        <path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('codeBlock') ? 'bg-background text-foreground shadow-xs' : ''}"
      onclick={() => editor.chain().focus().toggleCodeBlock().run()}
      title="Code Block"
      aria-label="Code Block"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polyline points="16 18 22 12 16 6" />
        <polyline points="8 6 2 12 8 18" />
      </svg>
    </button>
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer"
      onclick={() => editor.chain().focus().setHorizontalRule().run()}
      title="Divider Line"
      aria-label="Divider Line"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="3" y1="12" x2="21" y2="12" />
      </svg>
    </button>

    <span class="w-[1px] h-4 bg-border mx-1"></span>

    <!-- Clear formatting -->
    <button
      type="button"
      class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer"
      onclick={() => editor.chain().focus().clearNodes().unsetAllMarks().run()}
      title="Clear formatting"
      aria-label="Clear formatting"
    >
      <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="4" y1="4" x2="20" y2="20" />
        <path d="M4 7h16" />
        <path d="M10 11 8 17" />
        <path d="M14 11l2 6" />
      </svg>
    </button>
  </div>

  <!-- Link Input Toolbar -->
  {#if showLinkInput}
    <div class="flex items-center gap-2 p-2 bg-accent/40 border-t border-border text-xs animate-in fade-in duration-100">
      <svg class="size-3.5 text-muted-foreground ml-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
        <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
      </svg>
      <input
        type="url"
        placeholder="https://example.com"
        bind:value={linkUrl}
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault();
            applyLink();
          } else if (e.key === 'Escape') {
            showLinkInput = false;
          }
        }}
        class="h-7 flex-1 rounded-md border border-input bg-background px-2.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring shadow-xs"
      />
      <button
        type="button"
        class="h-7 px-2.5 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-xs font-medium cursor-pointer shadow-xs"
        onclick={applyLink}
      >
        Apply
      </button>
      {#if editor.isActive('link')}
        <button
          type="button"
          class="h-7 px-2 rounded-md border border-input bg-background hover:bg-destructive/10 hover:text-destructive text-xs cursor-pointer shadow-xs"
          onclick={removeLink}
        >
          Unlink
        </button>
      {/if}
      <button
        type="button"
        class="size-7 inline-flex items-center justify-center rounded text-muted-foreground hover:text-foreground cursor-pointer"
        onclick={() => { showLinkInput = false; }}
        title="Close"
        aria-label="Close"
      >
        <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 6 6 18M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}
</div>

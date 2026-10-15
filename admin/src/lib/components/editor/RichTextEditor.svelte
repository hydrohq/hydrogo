<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Editor } from '@tiptap/core';
  import StarterKit from '@tiptap/starter-kit';
  import Link from '@tiptap/extension-link';

  interface Props {
    value?: string;
    label?: string;
    error?: string;
    placeholder?: string;
  }

  let {
    value = $bindable(),
    label,
    error,
    placeholder = 'Start writing your story or content here…',
  }: Props = $props();

  let element: HTMLDivElement | null = $state(null);
  let editor = $state<Editor | null>(null);

  // Link popover state
  let showLinkInput = $state(false);
  let linkUrl = $state('');

  // Status counters
  let wordCount = $state(0);
  let charCount = $state(0);
  let currentBlockType = $state('Paragraph');

  function updateMetrics(ed: Editor) {
    const text = ed.getText();
    charCount = text.length;
    wordCount = text.trim() ? text.trim().split(/\s+/).length : 0;

    if (ed.isActive('heading', { level: 1 })) {
      currentBlockType = 'Heading 1';
    } else if (ed.isActive('heading', { level: 2 })) {
      currentBlockType = 'Heading 2';
    } else if (ed.isActive('heading', { level: 3 })) {
      currentBlockType = 'Heading 3';
    } else if (ed.isActive('blockquote')) {
      currentBlockType = 'Blockquote';
    } else if (ed.isActive('codeBlock')) {
      currentBlockType = 'Code Block';
    } else if (ed.isActive('bulletList')) {
      currentBlockType = 'Bullet List';
    } else if (ed.isActive('orderedList')) {
      currentBlockType = 'Numbered List';
    } else {
      currentBlockType = 'Paragraph';
    }
  }

  onMount(() => {
    if (!element) return;

    editor = new Editor({
      element: element,
      extensions: [
        StarterKit.configure({
          heading: {
            levels: [1, 2, 3],
          },
          link: false,
        }),
        Link.configure({
          openOnClick: false,
          HTMLAttributes: {
            class: 'text-primary underline underline-offset-2 hover:opacity-80 transition-opacity',
          },
        }),
      ],
      content: value || '',
      editorProps: {
        attributes: {
          class:
            'prose dark:prose-invert max-w-none focus:outline-none min-h-[180px] p-4 text-xs sm:text-sm text-foreground leading-relaxed',
        },
      },
      onUpdate: ({ editor: ed }) => {
        value = ed.getHTML();
        updateMetrics(ed);
      },
      onSelectionUpdate: ({ editor: ed }) => {
        updateMetrics(ed);
      },
    });

    if (editor) {
      updateMetrics(editor);
    }
  });

  onDestroy(() => {
    if (editor) {
      editor.destroy();
    }
  });

  $effect(() => {
    if (editor && value !== editor.getHTML()) {
      const isSame = editor.getHTML() === value;
      if (!isSame && !editor.isFocused) {
        editor.commands.setContent(value || '', { emitUpdate: false });
        updateMetrics(editor);
      }
    }
  });

  function openLinkPrompt() {
    if (!editor) return;
    const previousUrl = editor.getAttributes('link').href || '';
    linkUrl = previousUrl;
    showLinkInput = true;
  }

  function applyLink() {
    if (!editor) return;
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
    if (!editor) return;
    editor.chain().focus().unsetLink().run();
    showLinkInput = false;
  }
</script>

<div class="flex flex-col gap-1.5 w-full">
  {#if label}
    <div class="flex items-center justify-between">
      <span class="text-xs font-semibold text-foreground">
        {label}
      </span>
      <span class="text-[11px] text-muted-foreground font-mono">
        {currentBlockType}
      </span>
    </div>
  {/if}

  <div
    class="rounded-xl border {error
      ? 'border-destructive ring-1 ring-destructive'
      : 'border-border'} bg-card overflow-hidden shadow-xs transition-all focus-within:border-ring focus-within:ring-1 focus-within:ring-ring flex flex-col"
  >
    <!-- Toolbar -->
    {#if editor}
      <div
        class="flex flex-wrap items-center gap-0.5 p-1.5 border-b border-border bg-muted/40 text-muted-foreground select-none"
      >
        <!-- History: Undo / Redo -->
        <button
          type="button"
          disabled={!editor.can().undo()}
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground disabled:opacity-30 disabled:pointer-events-none transition-colors cursor-pointer"
          onclick={() => editor?.chain().focus().undo().run()}
          title="Undo (Ctrl+Z)"
          aria-label="Undo"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 7v6h6"></path>
            <path d="M21 17a9 9 0 0 0-9-9 9 9 0 0 0-6 2.3L3 13"></path>
          </svg>
        </button>

        <button
          type="button"
          disabled={!editor.can().redo()}
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground disabled:opacity-30 disabled:pointer-events-none transition-colors cursor-pointer"
          onclick={() => editor?.chain().focus().redo().run()}
          title="Redo (Ctrl+Y)"
          aria-label="Redo"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 7v6h-6"></path>
            <path d="M3 17a9 9 0 0 1 9-9 9 9 0 0 1 6 2.3l3 2.7"></path>
          </svg>
        </button>

        <span class="w-[1px] h-4 bg-border mx-1"></span>

        <!-- Headings -->
        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('paragraph') && !editor.isActive('heading') ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
          onclick={() => editor?.chain().focus().setParagraph().run()}
          title="Normal Text (Paragraph)"
        >
          P
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('heading', { level: 1 }) ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 1 }).run()}
          title="Heading 1"
        >
          H1
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('heading', { level: 2 }) ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()}
          title="Heading 2"
        >
          H2
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md text-xs font-semibold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('heading', { level: 3 }) ? 'bg-background text-foreground shadow-xs font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 3 }).run()}
          title="Heading 3"
        >
          H3
        </button>

        <span class="w-[1px] h-4 bg-border mx-1"></span>

        <!-- Inline Formatting -->
        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md font-bold hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('bold') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleBold().run()}
          title="Bold (Ctrl+B)"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M6 12h9a4 4 0 0 1 0 8H6v-8Z"></path>
            <path d="M6 4h8a4 4 0 0 1 0 8H6V4Z"></path>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md italic hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('italic') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleItalic().run()}
          title="Italic (Ctrl+I)"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="19" y1="4" x2="10" y2="4"></line>
            <line x1="14" y1="20" x2="5" y2="20"></line>
            <line x1="15" y1="4" x2="9" y2="20"></line>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('strike') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleStrike().run()}
          title="Strikethrough"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M16 4H9a3 3 0 0 0-2.83 4"></path>
            <path d="M14 12a4 4 0 0 1 0 8H6"></path>
            <line x1="4" y1="12" x2="20" y2="12"></line>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md font-mono text-xs hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('code') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleCode().run()}
          title="Inline Code"
        >
          &lt;/&gt;
        </button>

        <!-- Link Button -->
        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('link') ? 'bg-background text-primary shadow-xs' : ''}"
          onclick={openLinkPrompt}
          title="Insert / Edit Link"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
          </svg>
        </button>

        <span class="w-[1px] h-4 bg-border mx-1"></span>

        <!-- Lists & Blocks -->
        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('bulletList') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleBulletList().run()}
          title="Bullet List"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="8" y1="6" x2="21" y2="6"></line>
            <line x1="8" y1="12" x2="21" y2="12"></line>
            <line x1="8" y1="18" x2="21" y2="18"></line>
            <line x1="3" y1="6" x2="3.01" y2="6"></line>
            <line x1="3" y1="12" x2="3.01" y2="12"></line>
            <line x1="3" y1="18" x2="3.01" y2="18"></line>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('orderedList') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleOrderedList().run()}
          title="Numbered List"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="10" y1="6" x2="21" y2="6"></line>
            <line x1="10" y1="12" x2="21" y2="12"></line>
            <line x1="10" y1="18" x2="21" y2="18"></line>
            <path d="M4 6h1v4"></path>
            <path d="M4 10h2"></path>
            <path d="M6 18H4c0-1 2-2 2-3s-1-1.5-2-1"></path>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('blockquote') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleBlockquote().run()}
          title="Blockquote"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 21c3 0 7-1 7-8V5c0-1.25-.756-2.017-2-2H4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z"></path>
            <path d="M15 21c3 0 7-1 7-8V5c0-1.25-.757-2.017-2-2h-4c-1.25 0-2 .75-2 1.972V11c0 1.25.75 2 2 2 1 0 1 0 1 1v1c0 1-1 2-2 2s-1 .008-1 1.031V20c0 1 0 1 1 1z"></path>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer {editor.isActive('codeBlock') ? 'bg-background text-foreground shadow-xs' : ''}"
          onclick={() => editor?.chain().focus().toggleCodeBlock().run()}
          title="Code Block"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="16 18 22 12 16 6"></polyline>
            <polyline points="8 6 2 12 8 18"></polyline>
          </svg>
        </button>

        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer"
          onclick={() => editor?.chain().focus().setHorizontalRule().run()}
          title="Horizontal Rule"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="3" y1="12" x2="21" y2="12"></line>
          </svg>
        </button>

        <span class="w-[1px] h-4 bg-border mx-1"></span>

        <!-- Clear formatting -->
        <button
          type="button"
          class="size-7 inline-flex items-center justify-center rounded-md hover:bg-accent hover:text-foreground transition-colors cursor-pointer"
          onclick={() => editor?.chain().focus().clearNodes().unsetAllMarks().run()}
          title="Clear formatting"
        >
          <svg class="size-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="4" y1="4" x2="20" y2="20"></line>
            <path d="M4 7h16"></path>
            <path d="M10 11 8 17"></path>
            <path d="M14 11l2 6"></path>
          </svg>
        </button>
      </div>

      <!-- Link Inline Input Toolbar -->
      {#if showLinkInput}
        <div
          class="flex items-center gap-2 p-2 bg-accent/40 border-b border-border text-xs animate-in fade-in duration-100"
        >
          <svg class="size-3.5 text-muted-foreground ml-1" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
          </svg>
          <input
            type="url"
            placeholder="https://example.com"
            bind:value={linkUrl}
            onkeydown={(e) => { if (e.key === 'Enter') { e.preventDefault(); applyLink(); } else if (e.key === 'Escape') { showLinkInput = false; } }}
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
          >
            ✕
          </button>
        </div>
      {/if}
    {/if}

    <!-- Content Writing Area -->
    <div
      bind:this={element}
      class="editor-content flex-1 bg-background"
      data-placeholder={placeholder}
    ></div>

    <!-- Status Bar Footer -->
    <div
      class="flex items-center justify-between px-3 py-1.5 border-t border-border bg-muted/20 text-[11px] text-muted-foreground font-mono"
    >
      <div class="flex items-center gap-3">
        <span>{wordCount} words</span>
        <span>•</span>
        <span>{charCount} characters</span>
      </div>

      <div class="flex items-center gap-1.5">
        <span>~{Math.max(1, Math.ceil(wordCount / 200))} min read</span>
      </div>
    </div>
  </div>

  {#if error}
    <p class="text-xs text-destructive">{error}</p>
  {/if}
</div>

<style>
  :global(.editor-content .tiptap) {
    min-height: 200px;
  }
  :global(.editor-content .tiptap:focus) {
    outline: none;
  }
  :global(.editor-content .tiptap p) {
    margin-top: 0.5rem;
    margin-bottom: 0.5rem;
  }
  :global(.editor-content .tiptap h1) {
    font-size: 1.5rem;
    font-weight: 700;
    letter-spacing: -0.025em;
    margin-top: 1.25rem;
    margin-bottom: 0.5rem;
  }
  :global(.editor-content .tiptap h2) {
    font-size: 1.25rem;
    font-weight: 600;
    letter-spacing: -0.02em;
    margin-top: 1rem;
    margin-bottom: 0.375rem;
  }
  :global(.editor-content .tiptap h3) {
    font-size: 1.1rem;
    font-weight: 600;
    margin-top: 0.75rem;
    margin-bottom: 0.25rem;
  }
  :global(.editor-content .tiptap ul) {
    list-style-type: disc;
    padding-left: 1.25rem;
    margin-top: 0.5rem;
    margin-bottom: 0.5rem;
  }
  :global(.editor-content .tiptap ol) {
    list-style-type: decimal;
    padding-left: 1.25rem;
    margin-top: 0.5rem;
    margin-bottom: 0.5rem;
  }
  :global(.editor-content .tiptap blockquote) {
    border-left: 3px solid var(--border, #e4e4e7);
    padding-left: 0.875rem;
    margin-top: 0.75rem;
    margin-bottom: 0.75rem;
    font-style: italic;
    color: var(--muted-foreground, #71717a);
  }
  :global(.editor-content .tiptap code) {
    background-color: var(--muted, #f4f4f5);
    padding: 0.125rem 0.25rem;
    border-radius: 0.25rem;
    font-family: monospace;
    font-size: 0.8em;
  }
  :global(.editor-content .tiptap pre) {
    background-color: #18181b;
    color: #fafafa;
    padding: 0.75rem;
    border-radius: 0.5rem;
    font-family: monospace;
    font-size: 0.75rem;
    margin-top: 0.75rem;
    margin-bottom: 0.75rem;
    overflow-x: auto;
  }
  :global(.editor-content .tiptap pre code) {
    background: transparent;
    color: inherit;
    padding: 0;
  }
  :global(.editor-content .tiptap hr) {
    border: none;
    border-top: 1px solid var(--border, #e4e4e7);
    margin: 1.25rem 0;
  }
</style>

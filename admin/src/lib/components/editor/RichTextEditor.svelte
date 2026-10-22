<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Editor } from '@tiptap/core';
  import StarterKit from '@tiptap/starter-kit';
  import Link from '@tiptap/extension-link';
  import Placeholder from '@tiptap/extension-placeholder';
  import EditorToolbar from './EditorToolbar.svelte';
  import BubbleMenu from './BubbleMenu.svelte';
  import SlashCommandMenu, { type SlashItem } from './SlashCommandMenu.svelte';
  import { normalizeRichText } from './deltaToHtml';
  import { t, type MessageKey } from '$lib/i18n/index.svelte';

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
    placeholder,
  }: Props = $props();

  let element: HTMLDivElement | null = $state(null);
  let editor = $state<Editor | null>(null);

  // Status counters
  let wordCount = $state(0);
  let charCount = $state(0);
  let currentBlockType = $state<MessageKey>('editor.paragraphShort');

  // Floating Slash Command Palette state
  let showSlashMenu = $state(false);
  let slashMenuPos = $state<{ top: number; left: number }>({ top: 0, left: 0 });
  let slashQuery = $state('');
  let selectedSlashIndex = $state(0);

  // Floating Selection Bubble Menu state
  let showBubbleMenu = $state(false);
  let bubbleMenuPos = $state<{ top: number; left: number }>({ top: 0, left: 0 });

  const rawSlashCommands = $derived<Array<Omit<SlashItem, 'action'> & { action: (ed: Editor) => void }>>([
    {
      id: 'text',
      title: t('slash.text'),
      description: t('slash.textDesc'),
      badge: 'P',
      action: (ed) => ed.chain().focus().setParagraph().run(),
    },
    {
      id: 'h1',
      title: t('slash.h1'),
      description: t('slash.h1Desc'),
      badge: 'H1',
      action: (ed) => ed.chain().focus().toggleHeading({ level: 1 }).run(),
    },
    {
      id: 'h2',
      title: t('slash.h2'),
      description: t('slash.h2Desc'),
      badge: 'H2',
      action: (ed) => ed.chain().focus().toggleHeading({ level: 2 }).run(),
    },
    {
      id: 'h3',
      title: t('slash.h3'),
      description: t('slash.h3Desc'),
      badge: 'H3',
      action: (ed) => ed.chain().focus().toggleHeading({ level: 3 }).run(),
    },
    {
      id: 'bullet',
      title: t('slash.bullet'),
      description: t('slash.bulletDesc'),
      badge: 'List',
      action: (ed) => ed.chain().focus().toggleBulletList().run(),
    },
    {
      id: 'number',
      title: t('slash.numbered'),
      description: t('slash.numberedDesc'),
      badge: '1.',
      action: (ed) => ed.chain().focus().toggleOrderedList().run(),
    },
    {
      id: 'quote',
      title: t('slash.quote'),
      description: t('slash.quoteDesc'),
      badge: 'Quote',
      action: (ed) => ed.chain().focus().toggleBlockquote().run(),
    },
    {
      id: 'code',
      title: t('slash.code'),
      description: t('slash.codeDesc'),
      badge: '</>',
      action: (ed) => ed.chain().focus().toggleCodeBlock().run(),
    },
    {
      id: 'divider',
      title: t('slash.divider'),
      description: t('slash.dividerDesc'),
      badge: 'HR',
      action: (ed) => ed.chain().focus().setHorizontalRule().run(),
    },
  ]);

  const filteredSlashCommands = $derived.by(() => {
    if (!editor) return [];
    const q = slashQuery.toLowerCase().trim();
    const list = q
      ? rawSlashCommands.filter(
          (cmd) =>
            cmd.id.includes(q) ||
            cmd.title.toLowerCase().includes(q) ||
            cmd.description.toLowerCase().includes(q)
        )
      : rawSlashCommands;

    return list.map((cmd) => ({
      ...cmd,
      action: () => executeSlashCommand(cmd),
    }));
  });

  function updateMetrics(ed: Editor) {
    const text = ed.getText();
    charCount = text.length;
    wordCount = text.trim() ? text.trim().split(/\s+/).length : 0;

    if (ed.isActive('heading', { level: 1 })) {
      currentBlockType = 'editor.h1';
    } else if (ed.isActive('heading', { level: 2 })) {
      currentBlockType = 'editor.h2';
    } else if (ed.isActive('heading', { level: 3 })) {
      currentBlockType = 'editor.h3';
    } else if (ed.isActive('blockquote')) {
      currentBlockType = 'editor.quote';
    } else if (ed.isActive('codeBlock')) {
      currentBlockType = 'editor.codeBlock';
    } else if (ed.isActive('bulletList')) {
      currentBlockType = 'editor.bulletList';
    } else if (ed.isActive('orderedList')) {
      currentBlockType = 'editor.numberedList';
    } else {
      currentBlockType = 'editor.paragraphShort';
    }
  }

  function checkSlashAndBubbleMenu(ed: Editor) {
    if (!element) return;
    const { from, to, empty } = ed.state.selection;

    // 1. Check Floating Selection Bubble Menu
    if (!empty && from !== to) {
      showSlashMenu = false;
      try {
        const startCoords = ed.view.coordsAtPos(from);
        const endCoords = ed.view.coordsAtPos(to);
        const elRect = element.getBoundingClientRect();

        const top = startCoords.top - elRect.top - 46;
        const left = Math.max(
          10,
          Math.min(
            elRect.width - 260,
            (startCoords.left + endCoords.left) / 2 - elRect.left - 120
          )
        );

        bubbleMenuPos = { top, left };
        showBubbleMenu = true;
      } catch {
        showBubbleMenu = false;
      }
    } else {
      showBubbleMenu = false;
    }

    // 2. Check Slash Command Menu trigger
    if (empty) {
      const anchorFrom = ed.state.selection.$from;
      const textBefore = anchorFrom.parent.textBetween(0, anchorFrom.parentOffset, '\0', '\0');

      if (textBefore.startsWith('/')) {
        slashQuery = textBefore.slice(1);
        selectedSlashIndex = 0;

        try {
          const coords = ed.view.coordsAtPos(from);
          const elRect = element.getBoundingClientRect();

          let top = coords.bottom - elRect.top + 8;
          let left = Math.max(8, Math.min(elRect.width - 280, coords.left - elRect.left));

          // If near bottom of container, flip upwards
          if (top > elRect.height - 200 && top > 220) {
            top = coords.top - elRect.top - 240;
          }

          slashMenuPos = { top, left };
          showSlashMenu = true;
          return;
        } catch {
          showSlashMenu = false;
        }
      } else {
        showSlashMenu = false;
      }
    } else {
      showSlashMenu = false;
    }
  }

  function executeSlashCommand(cmd: { action: (ed: Editor) => void }) {
    if (!editor) return;
    const { from } = editor.state.selection;
    const anchorFrom = editor.state.selection.$from;
    const start = anchorFrom.start();

    // Delete the slash trigger text (e.g. "/h1" or "/")
    editor.chain().focus().deleteRange({ from: start, to: from }).run();
    cmd.action(editor);
    showSlashMenu = false;
    slashQuery = '';
  }

  onMount(() => {
    if (!element) return;

    const initialContent = normalizeRichText(value);
    if (value && initialContent && value !== initialContent) {
      value = initialContent;
    }

    editor = new Editor({
      element: element,
      // The page's content security policy refuses a style element added at
      // runtime, so ProseMirror's core rules live in app.css instead.
      injectCSS: false,
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
        Placeholder.configure({
          placeholder: ({ node }) => {
            if (node.type.name === 'heading') {
              return 'Heading...';
            }
            if (node.type.name === 'codeBlock') {
              return '// Write code here...';
            }
            if (node.type.name === 'blockquote') {
              return 'Empty quote...';
            }
            return placeholder ?? t('editor.placeholder');
          },
        }),
      ],
      content: initialContent || '',
      editorProps: {
        attributes: {
          class:
            'prose dark:prose-invert max-w-none focus:outline-none min-h-[220px] p-4 text-xs sm:text-sm text-foreground leading-relaxed',
        },
        handleKeyDown: (_view, event) => {
          if (showSlashMenu && filteredSlashCommands.length > 0) {
            if (event.key === 'ArrowDown') {
              event.preventDefault();
              selectedSlashIndex = (selectedSlashIndex + 1) % filteredSlashCommands.length;
              return true;
            }
            if (event.key === 'ArrowUp') {
              event.preventDefault();
              selectedSlashIndex =
                (selectedSlashIndex - 1 + filteredSlashCommands.length) %
                filteredSlashCommands.length;
              return true;
            }
            if (event.key === 'Enter') {
              event.preventDefault();
              filteredSlashCommands[selectedSlashIndex].action();
              return true;
            }
            if (event.key === 'Escape') {
              event.preventDefault();
              showSlashMenu = false;
              return true;
            }
          }
          return false;
        },
      },
      onUpdate: ({ editor: ed }) => {
        value = ed.getHTML();
        updateMetrics(ed);
        checkSlashAndBubbleMenu(ed);
      },
      onSelectionUpdate: ({ editor: ed }) => {
        updateMetrics(ed);
        checkSlashAndBubbleMenu(ed);
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
    if (editor && value !== undefined) {
      const normalized = normalizeRichText(value);
      if (normalized !== editor.getHTML() && !editor.isFocused) {
        editor.commands.setContent(normalized, { emitUpdate: false });
        if (value !== normalized) {
          value = normalized;
        }
        updateMetrics(editor);
      }
    }
  });
</script>

<div class="flex flex-col gap-1.5 w-full">
  {#if label}
    <div class="flex items-center justify-between">
      <span class="text-xs font-semibold text-foreground">
        {label}
      </span>
      <div class="flex items-center gap-2">
        <span class="inline-flex items-center gap-1 text-[10px] text-muted-foreground font-mono bg-muted/50 px-1.5 py-0.5 rounded border border-border">
          <kbd class="font-bold">/</kbd> for commands
        </span>
        <span class="text-[11px] text-muted-foreground font-mono">
          {t(currentBlockType)}
        </span>
      </div>
    </div>
  {/if}

  <div
    class="relative rounded-xl border {error
      ? 'border-destructive ring-1 ring-destructive'
      : 'border-border'} bg-card shadow-xs transition-all focus-within:border-ring focus-within:ring-1 focus-within:ring-ring flex flex-col"
  >
    <!-- Top Toolbar -->
    {#if editor}
      <EditorToolbar {editor} />
    {/if}

    <!-- Content Writing Canvas -->
    <div
      bind:this={element}
      class="editor-content flex-1 bg-background relative"
    >
      <!-- Floating Selection Bubble Menu (Notion-style) -->
      {#if showBubbleMenu && editor}
        <BubbleMenu
          {editor}
          position={bubbleMenuPos}
          onOpenLink={() => {
            const previousUrl = editor?.getAttributes('link').href || '';
            const nextUrl = window.prompt('Enter link URL:', previousUrl);
            if (nextUrl !== null) {
              if (nextUrl.trim() === '') {
                editor?.chain().focus().unsetLink().run();
              } else {
                editor?.chain().focus().setLink({ href: nextUrl.trim() }).run();
              }
            }
          }}
        />
      {/if}

      <!-- Floating Slash Commands Palette (Notion-style) -->
      {#if showSlashMenu && filteredSlashCommands.length > 0}
        <SlashCommandMenu
          items={filteredSlashCommands}
          selectedIndex={selectedSlashIndex}
          position={slashMenuPos}
          onSelect={(cmd) => cmd.action()}
          onHover={(index) => { selectedSlashIndex = index; }}
        />
      {/if}
    </div>

    <!-- Status Bar Footer -->
    <div
      class="flex items-center justify-between px-3 py-1.5 border-t border-border bg-muted/20 text-[11px] text-muted-foreground font-mono"
    >
      <div class="flex items-center gap-3">
        <span>{t('editor.words', { count: wordCount })}</span>
        <span>•</span>
        <span>{t('editor.chars', { count: charCount })}</span>
      </div>

      <div class="flex items-center gap-2">
        <span class="hidden sm:inline text-muted-foreground/70">{t('editor.tip')}</span>
        <span class="hidden sm:inline">•</span>
        <span>{t('editor.minRead', { minutes: Math.max(1, Math.ceil(wordCount / 200)) })}</span>
      </div>
    </div>
  </div>

  {#if error}
    <p class="text-xs text-destructive">{error}</p>
  {/if}
</div>

<style>
  :global(.editor-content .tiptap) {
    min-height: 220px;
  }
  :global(.editor-content .tiptap:focus) {
    outline: none;
  }
  :global(.editor-content .tiptap p) {
    margin-top: 0.5rem;
    margin-bottom: 0.5rem;
  }
  :global(.editor-content .tiptap h1) {
    font-size: 1.625rem;
    font-weight: 700;
    letter-spacing: -0.025em;
    margin-top: 1.5rem;
    margin-bottom: 0.5rem;
    color: hsl(var(--foreground));
  }
  :global(.editor-content .tiptap h2) {
    font-size: 1.35rem;
    font-weight: 600;
    letter-spacing: -0.02em;
    margin-top: 1.25rem;
    margin-bottom: 0.375rem;
    color: hsl(var(--foreground));
  }
  :global(.editor-content .tiptap h3) {
    font-size: 1.15rem;
    font-weight: 600;
    margin-top: 1rem;
    margin-bottom: 0.25rem;
    color: hsl(var(--foreground));
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
  /* Notion-style Quote / Callout */
  :global(.editor-content .tiptap blockquote) {
    border-left: 3px solid hsl(var(--primary));
    background-color: hsl(var(--muted) / 0.35);
    padding: 0.75rem 1rem;
    border-radius: 0 0.5rem 0.5rem 0;
    margin: 0.875rem 0;
    font-style: normal;
    color: hsl(var(--foreground));
  }
  /* Notion-style Inline Code */
  :global(.editor-content .tiptap code) {
    background-color: hsl(var(--muted));
    color: hsl(var(--primary));
    padding: 0.15rem 0.35rem;
    border-radius: 0.25rem;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.85em;
    border: 1px solid hsl(var(--border) / 0.5);
  }
  /* Notion-style Code Block */
  :global(.editor-content .tiptap pre) {
    background-color: #09090b;
    color: #f4f4f5;
    padding: 1rem;
    border-radius: 0.625rem;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.8rem;
    margin-top: 0.875rem;
    margin-bottom: 0.875rem;
    overflow-x: auto;
    border: 1px solid hsl(var(--border) / 0.4);
    box-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1);
  }
  :global(.editor-content .tiptap pre code) {
    background: transparent;
    color: inherit;
    padding: 0;
    border: none;
  }
  :global(.editor-content .tiptap hr) {
    border: none;
    border-top: 1px solid hsl(var(--border));
    margin: 1.5rem 0;
  }
  /* Notion-style Empty Placeholders */
  :global(.editor-content .tiptap p.is-editor-empty:first-child::before),
  :global(.editor-content .tiptap p.is-empty::before) {
    content: attr(data-placeholder);
    float: left;
    color: hsl(var(--muted-foreground));
    opacity: 0.45;
    pointer-events: none;
    height: 0;
  }
  :global(.editor-content .tiptap h1.is-empty::before),
  :global(.editor-content .tiptap h2.is-empty::before),
  :global(.editor-content .tiptap h3.is-empty::before) {
    content: attr(data-placeholder);
    float: left;
    color: hsl(var(--muted-foreground));
    opacity: 0.35;
    pointer-events: none;
    height: 0;
  }
</style>

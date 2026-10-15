<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Editor } from '@tiptap/core';
  import StarterKit from '@tiptap/starter-kit';

  interface Props {
    value?: string;
    label?: string;
    error?: string;
    placeholder?: string;
  }

  let {
    value = $bindable(''),
    label,
    error,
    placeholder = 'Start writing…'
  }: Props = $props();

  let element: HTMLDivElement | null = $state(null);
  let editor: Editor | null = $state(null);

  onMount(() => {
    if (!element) return;

    editor = new Editor({
      element: element,
      extensions: [
        StarterKit,
      ],
      content: value || '',
      editorProps: {
        attributes: {
          class: 'prose dark:prose-invert max-w-none focus:outline-none min-h-[140px] px-3 py-2 text-xs text-foreground',
        },
      },
      onUpdate: ({ editor }) => {
        value = editor.getHTML();
      },
    });
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
      }
    }
  });
</script>

<div class="flex flex-col gap-1.5 w-full">
  {#if label}
    <span class="text-xs font-medium text-foreground">
      {label}
    </span>
  {/if}

  <div class="rounded-md border {error ? 'border-destructive' : 'border-input'} bg-background overflow-hidden focus-within:ring-1 focus-within:ring-ring shadow-xs transition-colors">
    {#if editor}
      <div class="flex items-center gap-1 p-1 border-b border-border bg-muted/50 text-muted-foreground text-xs">
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground {editor.isActive('bold') ? 'bg-accent text-foreground font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleBold().run()}
        >
          B
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground italic {editor.isActive('italic') ? 'bg-accent text-foreground' : ''}"
          onclick={() => editor?.chain().focus().toggleItalic().run()}
        >
          I
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground {editor.isActive('heading', { level: 2 }) ? 'bg-accent text-foreground font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()}
        >
          H2
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground {editor.isActive('heading', { level: 3 }) ? 'bg-accent text-foreground font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 3 }).run()}
        >
          H3
        </button>
        <span class="w-[1px] h-3.5 bg-border mx-1"></span>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground {editor.isActive('bulletList') ? 'bg-accent text-foreground' : ''}"
          onclick={() => editor?.chain().focus().toggleBulletList().run()}
        >
          List
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground {editor.isActive('orderedList') ? 'bg-accent text-foreground' : ''}"
          onclick={() => editor?.chain().focus().toggleOrderedList().run()}
        >
          1. 2.
        </button>
        <button
          type="button"
          class="px-2 py-0.5 rounded text-xs hover:bg-accent hover:text-foreground {editor.isActive('blockquote') ? 'bg-accent text-foreground' : ''}"
          onclick={() => editor?.chain().focus().toggleBlockquote().run()}
        >
          Quote
        </button>
      </div>
    {/if}

    <div bind:this={element} data-placeholder={placeholder}></div>
  </div>

  {#if error}
    <p class="text-xs text-destructive">{error}</p>
  {/if}
</div>

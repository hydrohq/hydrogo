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
          class: 'prose prose-invert max-w-none focus:outline-none min-h-[140px] px-3.5 py-2.5 text-sm text-slate-100',
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
        editor.commands.setContent(value || '', false);
      }
    }
  });
</script>

<div class="flex flex-col gap-1.5 w-full">
  {#if label}
    <span class="text-xs font-semibold uppercase tracking-wider text-slate-300">
      {label}
    </span>
  {/if}

  <div class="rounded-xl border {error ? 'border-rose-500' : 'border-slate-800'} bg-slate-900/90 overflow-hidden focus-within:ring-2 focus-within:ring-sky-500/20 focus-within:border-sky-500 transition-all">
    {#if editor}
      <div class="flex items-center gap-1 p-1.5 border-b border-slate-800/80 bg-slate-950/60 text-slate-400 text-xs">
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 {editor.isActive('bold') ? 'bg-slate-800 text-sky-400 font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleBold().run()}
        >
          B
        </button>
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 italic {editor.isActive('italic') ? 'bg-slate-800 text-sky-400' : ''}"
          onclick={() => editor?.chain().focus().toggleItalic().run()}
        >
          I
        </button>
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 {editor.isActive('heading', { level: 2 }) ? 'bg-slate-800 text-sky-400 font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 2 }).run()}
        >
          H2
        </button>
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 {editor.isActive('heading', { level: 3 }) ? 'bg-slate-800 text-sky-400 font-bold' : ''}"
          onclick={() => editor?.chain().focus().toggleHeading({ level: 3 }).run()}
        >
          H3
        </button>
        <span class="w-[1px] h-4 bg-slate-800 mx-1"></span>
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 {editor.isActive('bulletList') ? 'bg-slate-800 text-sky-400' : ''}"
          onclick={() => editor?.chain().focus().toggleBulletList().run()}
        >
          List
        </button>
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 {editor.isActive('orderedList') ? 'bg-slate-800 text-sky-400' : ''}"
          onclick={() => editor?.chain().focus().toggleOrderedList().run()}
        >
          1. 2.
        </button>
        <button
          type="button"
          class="px-2 py-1 rounded hover:bg-slate-800 hover:text-slate-100 {editor.isActive('blockquote') ? 'bg-slate-800 text-sky-400' : ''}"
          onclick={() => editor?.chain().focus().toggleBlockquote().run()}
        >
          Quote
        </button>
      </div>
    {/if}

    <div bind:this={element} data-placeholder={placeholder}></div>
  </div>

  {#if error}
    <p class="text-xs text-rose-400">{error}</p>
  {/if}
</div>

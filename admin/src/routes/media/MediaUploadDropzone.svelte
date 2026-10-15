<script lang="ts">

  interface Props {
    uploading: boolean;
    onupload: (files: FileList) => void;
  }

  let { uploading, onupload }: Props = $props();

  let isDragging = $state(false);
  let fileInput: HTMLInputElement | null = $state(null);

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    isDragging = true;
  }

  function handleDragLeave(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    if (e.dataTransfer?.files && e.dataTransfer.files.length > 0) {
      onupload(e.dataTransfer.files);
    }
  }

  function handleInputChange(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      onupload(target.files);
    }
  }
</script>

<div
  role="region"
  aria-label="File upload dropzone"
  class="relative rounded-2xl border-2 border-dashed p-6 transition-all duration-150 text-center flex flex-col items-center justify-center gap-2 {isDragging ? 'border-sky-500 bg-sky-950/20' : 'border-slate-800 bg-slate-900/40 hover:border-slate-700'}"
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
>
  <input
    type="file"
    class="hidden"
    multiple
    bind:this={fileInput}
    onchange={handleInputChange}
  />
  <div class="w-10 h-10 rounded-full bg-slate-800/80 flex items-center justify-center text-xl text-sky-400">
    📁
  </div>
  <div>
    <p class="text-sm font-semibold text-slate-200">
      Drag & drop assets here, or
      <button
        type="button"
        class="text-sky-400 hover:text-sky-300 underline font-medium cursor-pointer ml-1"
        onclick={() => fileInput?.click()}
      >
        browse files
      </button>
    </p>
    <p class="text-xs text-slate-500 mt-0.5">Images, PDFs, documents up to 50MB</p>
  </div>
  {#if uploading}
    <div class="flex items-center gap-2 text-xs text-sky-400 font-medium mt-1">
      <span class="animate-spin">⏳</span> Uploading files…
    </div>
  {/if}
</div>

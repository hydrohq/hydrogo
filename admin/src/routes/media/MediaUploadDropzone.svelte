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
  class="relative rounded-xl border border-dashed p-6 transition-all duration-150 text-center flex flex-col items-center justify-center gap-2 {isDragging ? 'border-primary bg-primary/5' : 'border-border bg-card hover:bg-muted/30 hover:border-muted-foreground/40'}"
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
  <div class="size-10 rounded-full bg-muted border border-border flex items-center justify-center text-foreground">
    <svg class="size-5 text-muted-foreground" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
      <polyline points="17 8 12 3 7 8"></polyline>
      <line x1="12" y1="3" x2="12" y2="15"></line>
    </svg>
  </div>
  <div>
    <p class="text-sm font-medium text-foreground">
      Drag & drop assets here, or
      <button
        type="button"
        class="text-foreground underline underline-offset-4 font-semibold cursor-pointer ml-1 hover:text-muted-foreground"
        onclick={() => fileInput?.click()}
      >
        browse files
      </button>
    </p>
    <p class="text-xs text-muted-foreground mt-0.5">Images, PDFs, documents up to 50MB</p>
  </div>
  {#if uploading}
    <div class="flex items-center gap-2 text-xs text-foreground font-medium mt-1">
      <svg class="size-3.5 animate-spin" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="12" y1="2" x2="12" y2="6"></line>
        <line x1="12" y1="18" x2="12" y2="22"></line>
        <line x1="4.93" y1="4.93" x2="7.76" y2="7.76"></line>
        <line x1="16.24" y1="16.24" x2="19.07" y2="19.07"></line>
        <line x1="2" y1="12" x2="6" y2="12"></line>
        <line x1="18" y1="12" x2="22" y2="12"></line>
        <line x1="4.93" y1="19.07" x2="7.76" y2="16.24"></line>
        <line x1="16.24" y1="7.76" x2="19.07" y2="4.93"></line>
      </svg>
      Uploading files…
    </div>
  {/if}
</div>

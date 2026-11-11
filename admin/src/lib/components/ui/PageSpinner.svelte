<script lang="ts">
  import { t } from '$lib/i18n/index.svelte';

  let { fullscreen = false, compact = false }: { fullscreen?: boolean; compact?: boolean } = $props();
</script>

<div
  class="flex w-full items-center justify-center {fullscreen ? 'min-h-screen' : compact ? 'min-h-[120px] py-6' : 'min-h-[40vh]'}"
  role="status"
  aria-live="polite"
  data-testid="page-spinner"
>
  <div class="size-7 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
  <span class="sr-only">{t('common.loading')}</span>
</div>

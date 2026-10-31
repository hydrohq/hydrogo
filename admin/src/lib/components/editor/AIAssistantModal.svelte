<script lang="ts">
  import Modal from '$lib/components/ui/Modal.svelte';
  import Button from '$lib/components/ui/Button.svelte';
  import { api } from '$lib/api/client';

  interface Props {
    open: boolean;
    pageId: string;
    currentTitle: string;
    currentBody?: string;
    currentLocale?: string;
    onapply: (title: string, body: string) => void;
    onclose: () => void;
  }

  let {
    open = false,
    pageId,
    currentTitle = '',
    currentBody = '',
    currentLocale = 'en',
    onapply,
    onclose,
  }: Props = $props();

  type Operation = 'refine' | 'seo' | 'summary' | 'translate';
  type Tone = 'professional' | 'concise' | 'friendly';

  let operation = $state<Operation>('refine');
  let tone = $state<Tone>('professional');
  let targetLocale = $state('en');
  let generating = $state(false);
  let applying = $state(false);
  let errorMsg = $state<string | null>(null);

  interface AISuggestion {
    title: string;
    body: string;
  }

  let suggestion = $state<AISuggestion | null>(null);

  async function handleGenerate() {
    generating = true;
    errorMsg = null;
    suggestion = null;

    try {
      const payload: Record<string, any> = {
        operation,
        tone,
        locale: currentLocale,
        consent: true,
      };

      if (operation === 'translate') {
        payload.target_locale = targetLocale;
      }

      const res = await api.post<{ suggestion?: AISuggestion; data?: { suggestion?: AISuggestion } }>(
        `/api/v1/admin/pages/${pageId}/ai-suggest`,
        payload
      );

      const result = res?.suggestion || res?.data?.suggestion;
      if (result) {
        suggestion = result;
      } else {
        errorMsg = 'No suggestion returned by the AI provider.';
      }
    } catch (err: any) {
      errorMsg = err?.message || 'Failed to generate AI suggestion. Check your AI provider settings.';
    } finally {
      generating = false;
    }
  }

  async function handleApply() {
    if (!suggestion) return;
    applying = true;
    try {
      // Record acceptance audit log in backend
      await api.post(`/api/v1/admin/pages/${pageId}/ai-applied`, {});
    } catch {
      // Non-fatal audit log attempt
    } finally {
      applying = false;
      onapply(suggestion.title, suggestion.body);
      onclose();
    }
  }
</script>

<Modal {open} title="✨ AI Content Assistant" size="xl" {onclose}>
  <div class="flex flex-col gap-4 text-sm">
    {#if errorMsg}
      <div class="p-3 text-xs rounded-md bg-destructive/10 text-destructive border border-destructive/20 flex items-center justify-between">
        <span>{errorMsg}</span>
        <button type="button" class="underline ml-2" onclick={() => (errorMsg = null)}>Dismiss</button>
      </div>
    {/if}

    <!-- Operation Tabs -->
    <div class="flex flex-col gap-1.5">
      <span class="text-xs font-medium text-muted-foreground">Select AI Capability</span>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
        <button
          type="button"
          onclick={() => { operation = 'refine'; suggestion = null; }}
          class={`flex flex-col items-center justify-center p-2.5 rounded-lg border text-xs font-medium transition-all ${
            operation === 'refine'
              ? 'border-primary bg-primary/10 text-primary shadow-xs'
              : 'border-border bg-card hover:bg-accent text-foreground'
          }`}
        >
          <span class="text-base mb-1">✨</span>
          <span>Polish & Refine</span>
        </button>

        <button
          type="button"
          onclick={() => { operation = 'seo'; suggestion = null; }}
          class={`flex flex-col items-center justify-center p-2.5 rounded-lg border text-xs font-medium transition-all ${
            operation === 'seo'
              ? 'border-primary bg-primary/10 text-primary shadow-xs'
              : 'border-border bg-card hover:bg-accent text-foreground'
          }`}
        >
          <span class="text-base mb-1">🔍</span>
          <span>SEO Optimize</span>
        </button>

        <button
          type="button"
          onclick={() => { operation = 'summary'; suggestion = null; }}
          class={`flex flex-col items-center justify-center p-2.5 rounded-lg border text-xs font-medium transition-all ${
            operation === 'summary'
              ? 'border-primary bg-primary/10 text-primary shadow-xs'
              : 'border-border bg-card hover:bg-accent text-foreground'
          }`}
        >
          <span class="text-base mb-1">📝</span>
          <span>Summarize</span>
        </button>

        <button
          type="button"
          onclick={() => { operation = 'translate'; suggestion = null; }}
          class={`flex flex-col items-center justify-center p-2.5 rounded-lg border text-xs font-medium transition-all ${
            operation === 'translate'
              ? 'border-primary bg-primary/10 text-primary shadow-xs'
              : 'border-border bg-card hover:bg-accent text-foreground'
          }`}
        >
          <span class="text-base mb-1">🌐</span>
          <span>Translate</span>
        </button>
      </div>
    </div>

    <!-- Configuration Options -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3 bg-muted/40 rounded-lg border border-border/50">
      <div class="flex flex-col gap-1">
        <label for="ai-tone" class="text-xs font-medium text-muted-foreground">Desired Tone</label>
        <select
          id="ai-tone"
          bind:value={tone}
          class="h-8 rounded-md border border-input bg-background px-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        >
          <option value="professional">Professional & Authoritative</option>
          <option value="concise">Concise & Direct</option>
          <option value="friendly">Engaging & Friendly</option>
        </select>
      </div>

      {#if operation === 'translate'}
        <div class="flex flex-col gap-1">
          <label for="ai-target-locale" class="text-xs font-medium text-muted-foreground">Target Language</label>
          <select
            id="ai-target-locale"
            bind:value={targetLocale}
            class="h-8 rounded-md border border-input bg-background px-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="en">English (US/UK)</option>
            <option value="tr">Turkish (Türkçe)</option>
            <option value="de">German (Deutsch)</option>
            <option value="fr">French (Français)</option>
            <option value="es">Spanish (Español)</option>
          </select>
        </div>
      {/if}
    </div>

    <!-- Trigger Button -->
    <div class="flex justify-end">
      <Button variant="primary" size="sm" loading={generating} onclick={handleGenerate}>
        {generating ? 'Processing with AI...' : 'Generate Suggestion'}
      </Button>
    </div>

    <!-- Suggestion Diff Preview -->
    {#if suggestion}
      <div class="flex flex-col gap-3 mt-2 border-t pt-3">
        <div class="flex items-center justify-between">
          <span class="text-xs font-semibold uppercase tracking-wider text-muted-foreground">AI Suggestion Preview</span>
          <Button variant="primary" size="sm" loading={applying} onclick={handleApply}>
            ✓ Apply to Editor
          </Button>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <!-- Before -->
          <div class="flex flex-col gap-2 p-3 bg-muted/20 border border-border/60 rounded-md">
            <span class="text-xs font-semibold text-muted-foreground">Current Version</span>
            <div class="text-xs font-medium text-foreground line-clamp-2">
              <span class="text-muted-foreground">Title:</span> {currentTitle || '(empty)'}
            </div>
            <div class="text-xs text-muted-foreground whitespace-pre-wrap max-h-48 overflow-y-auto">
              {currentBody || '(empty)'}
            </div>
          </div>

          <!-- After -->
          <div class="flex flex-col gap-2 p-3 bg-primary/5 border border-primary/20 rounded-md">
            <span class="text-xs font-semibold text-primary">✨ Suggested Version</span>
            <div class="text-xs font-medium text-foreground line-clamp-2">
              <span class="text-muted-foreground">Title:</span> {suggestion.title}
            </div>
            <div class="text-xs text-foreground whitespace-pre-wrap max-h-48 overflow-y-auto font-sans">
              {suggestion.body}
            </div>
          </div>
        </div>
      </div>
    {/if}
  </div>

  {#snippet footer()}
    <div class="flex items-center justify-between w-full">
      <span class="text-[11px] text-muted-foreground">
        Powered by HydroGo Provider-Neutral AI Engine
      </span>
      <Button variant="outline" size="sm" onclick={onclose}>Close</Button>
    </div>
  {/snippet}
</Modal>

<script lang="ts">
  import { api, ApiClientError } from '$lib/api/client';
  import type { CopilotActionLog, TenantGovernance } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';

  interface Props {
    open?: boolean;
    onclose?: () => void;
  }

  let { open = $bindable(false), onclose }: Props = $props();

  let actions = $state<CopilotActionLog[]>([]);
  let governance = $state<TenantGovernance | null>(null);
  let loading = $state(false);
  let sending = $state(false);
  let undoingId = $state('');
  let promptText = $state('');
  let errorMsg = $state('');
  let errorRequestId = $state('');

  const quickPrompts = [
    'Mağaza temasını koyu ve altın rengi yap',
    'İpek Gömlek fiyatını 1350 TL olarak güncelle',
    'Yeni ürün ekle: Keten Elbise, fiyat 1650 TL'
  ];

  $effect(() => {
    if (open) {
      loadHistory();
    }
  });

  async function loadHistory() {
    loading = true;
    errorMsg = '';
    errorRequestId = '';
    try {
      const res = await api.get<{ actions: CopilotActionLog[]; governance: TenantGovernance }>(
        '/api/v1/admin/ai/actions'
      );
      actions = res.actions ?? [];
      governance = res.governance ?? null;
    } catch (err) {
      if (err instanceof ApiClientError) {
        errorMsg = err.message;
        errorRequestId = err.requestId ?? '';
      } else {
        errorMsg = 'AI geçmişi yüklenemedi.';
      }
    } finally {
      loading = false;
    }
  }

  async function sendPrompt(e?: Event) {
    if (e) e.preventDefault();
    const trimmed = promptText.trim();
    if (!trimmed || sending) return;
    sending = true;
    errorMsg = '';
    errorRequestId = '';
    try {
      const res = await api.post<{
        reply: string;
        action_log: CopilotActionLog;
        governance: TenantGovernance;
      }>('/api/v1/admin/ai/copilot', { prompt: trimmed });
      promptText = '';
      if (res.action_log) {
        actions = [res.action_log, ...actions];
      }
      if (res.governance) {
        governance = res.governance;
      }
    } catch (err) {
      if (err instanceof ApiClientError) {
        errorMsg = err.message;
        errorRequestId = err.requestId ?? '';
      } else {
        errorMsg = 'AI isteği işlenirken hata oluştu.';
      }
    } finally {
      sending = false;
    }
  }

  async function handleUndo(actionId: string) {
    if (undoingId) return;
    undoingId = actionId;
    errorMsg = '';
    errorRequestId = '';
    try {
      const res = await api.post<{ action_log: CopilotActionLog }>(
        `/api/v1/admin/ai/actions/${encodeURIComponent(actionId)}/undo`,
        {}
      );
      actions = actions.map((item) => (item.id === actionId ? res.action_log : item));
    } catch (err) {
      if (err instanceof ApiClientError) {
        errorMsg = err.message;
        errorRequestId = err.requestId ?? '';
      } else {
        errorMsg = 'İşlem geri alınamadı.';
      }
    } finally {
      undoingId = '';
    }
  }

  function closeDrawer() {
    open = false;
    onclose?.();
  }
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex justify-end bg-black/40 backdrop-blur-xs">
    <div
      role="dialog"
      aria-modal="true"
      aria-label="AI Mağaza Asistanı (Copilot)"
      class="flex h-full w-full max-w-md flex-col border-l border-border bg-card text-card-foreground shadow-2xl"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-border px-4 py-3.5">
        <div class="flex items-center gap-2.5">
          <div class="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <svg class="size-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 3l1.9 5.8a2 2 0 0 0 1.3 1.3L21 12l-5.8 1.9a2 2 0 0 0-1.3 1.3L12 21l-1.9-5.8a2 2 0 0 0-1.3-1.3L3 12l5.8-1.9a2 2 0 0 0 1.3-1.3L12 3z" />
            </svg>
          </div>
          <div>
            <h2 class="text-sm font-semibold text-foreground">AI Mağaza Asistanı</h2>
            <p class="text-[11px] text-muted-foreground">
              {#if governance}
                {governance.ai_provider || 'gemini'} · {governance.ai_model || 'varsayılan'}
                {#if governance.has_custom_api_key}
                  <span class="text-emerald-600 dark:text-emerald-400">(BYOK)</span>
                {/if}
              {:else}
                Doğal dil ile ürün, fiyat ve tasarım yönetimi
              {/if}
            </p>
          </div>
        </div>
        <button
          type="button"
          onclick={closeDrawer}
          aria-label="Kapat"
          class="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground cursor-pointer"
        >
          ✕
        </button>
      </div>

      <!-- Quota Strip -->
      {#if governance && governance.ai_monthly_token_limit > 0}
        {@const pct = Math.min(100, Math.round((governance.ai_tokens_used / governance.ai_monthly_token_limit) * 100))}
        <div class="border-b border-border bg-muted/30 px-4 py-2 text-[11px]">
          <div class="flex items-center justify-between text-muted-foreground mb-1">
            <span>Aylık AI Token Kotası</span>
            <span class="font-mono">
              {governance.ai_tokens_used.toLocaleString('tr-TR')} / {governance.ai_monthly_token_limit.toLocaleString('tr-TR')} (%{pct})
            </span>
          </div>
          <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div class="h-full bg-primary transition-all" style="width: {pct}%"></div>
          </div>
        </div>
      {/if}

      {#if errorMsg}
        <div class="mx-4 mt-3 rounded-lg border border-destructive/40 bg-destructive/10 p-2.5 text-xs text-destructive">
          <div class="font-medium">{errorMsg}</div>
          {#if errorRequestId}
            <div class="mt-0.5 font-mono text-[10px] opacity-80">request_id: {errorRequestId}</div>
          {/if}
        </div>
      {/if}

      <!-- Action History Feed -->
      <div class="flex-1 space-y-3 overflow-y-auto p-4">
        {#if loading && actions.length === 0}
          <div class="py-8 text-center text-xs text-muted-foreground">Yükleniyor...</div>
        {:else if actions.length === 0}
          <div class="rounded-xl border border-dashed border-border p-5 text-center space-y-2">
            <p class="text-xs font-medium text-foreground">Henüz bir AI işlemi yok</p>
            <p class="text-[11px] text-muted-foreground">
              Mağazanızdaki ürünleri, fiyatları veya görünüm renklerini doğal dille değiştirmek için aşağıdan bir komut yazın.
            </p>
            <div class="flex flex-wrap justify-center gap-1.5 pt-2">
              {#each quickPrompts as chip}
                <button
                  type="button"
                  onclick={() => (promptText = chip)}
                  class="rounded-full border border-border bg-background px-2.5 py-1 text-[11px] text-muted-foreground hover:border-primary/40 hover:text-foreground cursor-pointer"
                >
                  {chip}
                </button>
              {/each}
            </div>
          </div>
        {:else}
          {#each actions as act (act.id)}
            <div class="rounded-xl border border-border bg-background p-3 space-y-2 shadow-2xs">
              <div class="flex items-start justify-between gap-2">
                <p class="text-xs font-medium text-foreground">“{act.prompt}”</p>
                <span class="shrink-0 font-mono text-[10px] text-muted-foreground">
                  {act.tokens_used} tok
                </span>
              </div>
              <p class="text-xs text-muted-foreground">{act.reply}</p>

              {#if act.operations && act.operations.length > 0}
                <div class="space-y-1 pt-1">
                  {#each act.operations as op}
                    <div class="flex items-center gap-1.5 rounded bg-muted/50 px-2 py-1 font-mono text-[10px] text-muted-foreground">
                      <span class="uppercase font-semibold text-foreground">{op.op}</span>
                      <span>{op.collection_id}</span>
                      {#if op.record_id}
                        <span class="truncate opacity-75">#{op.record_id.slice(0, 8)}</span>
                      {/if}
                    </div>
                  {/each}
                </div>

                <div class="flex items-center justify-between pt-1">
                  {#if act.undone}
                    <span class="text-[11px] font-medium text-amber-600 dark:text-amber-400">
                      ↺ Geri alındı
                    </span>
                  {:else}
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={undoingId === act.id}
                      onclick={() => handleUndo(act.id)}
                    >
                      {undoingId === act.id ? 'Geri Alınıyor...' : '↺ Geri Al (Undo)'}
                    </Button>
                  {/if}
                </div>
              {/if}
            </div>
          {/each}
        {/if}
      </div>

      <!-- Prompt Input Form -->
      <form onsubmit={sendPrompt} class="border-t border-border bg-background p-3 space-y-2">
        <textarea
          bind:value={promptText}
          rows="2"
          placeholder="Örn: İpek gömleğin fiyatını 1250 TL yap veya temayı koyu altın yap..."
          class="w-full resize-none rounded-lg border border-input bg-background px-3 py-2 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary"
        ></textarea>
        <div class="flex items-center justify-between">
          <span class="text-[10px] text-muted-foreground">Her işlem tek tıkla geri alınabilir.</span>
          <Button type="submit" size="sm" disabled={sending || !promptText.trim()}>
            {sending ? 'Uygulanıyor...' : 'Gönder'}
          </Button>
        </div>
      </form>
    </div>
  </div>
{/if}

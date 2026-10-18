<script lang="ts">
  import type { Comment } from '#lib';

  interface Props {
    slug: string;
    comments: Comment[];
    requireEmail?: boolean;
  }

  let { slug, comments = [], requireEmail = false }: Props = $props();

  let name = $state('');
  let email = $state('');
  let body = $state('');
  let status = $state<'idle' | 'submitting' | 'done' | 'error'>('idle');
  let errorMessage = $state('');

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!name.trim() || !body.trim() || (requireEmail && !email.trim())) {
      status = 'error';
      errorMessage = 'Lütfen gerekli alanları doldurun.';
      return;
    }
    status = 'submitting';
    try {
      const res = await fetch('/api/comments', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ slug, author_name: name, author_email: email, body }),
      });
      const json = await res.json();
      if (!res.ok) {
        status = 'error';
        errorMessage = json.message || 'Yorum gönderilemedi.';
        return;
      }
      status = 'done';
      name = '';
      email = '';
      body = '';
    } catch {
      status = 'error';
      errorMessage = 'Sunucuya bağlanılamadı.';
    }
  }

  function dateOf(c: Comment): string {
    return c.created_at ? new Date(c.created_at).toLocaleDateString('tr-TR') : '';
  }
</script>

<section class="mt-10 pt-8 border-t border-[var(--hp-border)]">
  <h3 class="text-lg md:text-xl font-bold text-[var(--hp-ink)] mb-5">
    Yorumlar {#if comments.length > 0}<span class="text-[var(--hp-muted)] font-normal">({comments.length})</span>{/if}
  </h3>

  {#if comments.length > 0}
    <div class="space-y-4 mb-8">
      {#each comments as c (c.id)}
        <div class="p-4 rounded-xl bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)]">
          <div class="flex items-center justify-between mb-1.5">
            <span class="text-sm font-bold text-[var(--hp-ink)]">{c.author_name}</span>
            <span class="text-[11px] text-[var(--hp-muted)] hp-tabular">{dateOf(c)}</span>
          </div>
          <p class="text-sm text-[var(--hp-ink-soft)] leading-relaxed whitespace-pre-line">{c.body}</p>
        </div>
      {/each}
    </div>
  {:else}
    <p class="text-sm text-[var(--hp-muted)] mb-8">Henüz yorum yapılmamış. İlk yorumu siz yazın.</p>
  {/if}

  {#if status === 'done'}
    <div class="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-sm font-medium">
      Yorumunuz alındı, moderasyon sonrası yayınlanacak.
    </div>
  {:else}
    <form onsubmit={submit} class="space-y-3 p-4 rounded-xl bg-[var(--hp-surface)] border border-[var(--hp-border)]">
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <input
          type="text"
          bind:value={name}
          placeholder="Adınız"
          required
          class="px-3 py-2 rounded-lg border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
        />
        <input
          type="email"
          bind:value={email}
          placeholder={requireEmail ? 'E-posta (zorunlu)' : 'E-posta (opsiyonel)'}
          required={requireEmail}
          class="px-3 py-2 rounded-lg border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
        />
      </div>
      <textarea
        bind:value={body}
        placeholder="Yorumunuz…"
        required
        rows="3"
        maxlength="2000"
        class="w-full px-3 py-2 rounded-lg border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors resize-y"
      ></textarea>
      {#if status === 'error'}
        <p class="text-xs text-[var(--hp-primary)]">{errorMessage}</p>
      {/if}
      <button
        type="submit"
        disabled={status === 'submitting'}
        class="px-4 py-2 rounded-lg bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] disabled:opacity-60 text-white text-sm font-bold transition-colors cursor-pointer"
      >
        {status === 'submitting' ? 'Gönderiliyor…' : 'Yorum Gönder'}
      </button>
    </form>
  {/if}
</section>

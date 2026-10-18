<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, sanitizeHtml, getCategoryBadge, fetchApi, slugify, type NewsArticle } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';
  import NewsCard from '#lib/components/NewsCard.svelte';
  import CommentsSection from '#lib/components/CommentsSection.svelte';
  import { onMount, tick } from 'svelte';

  let { data }: PageProps = $props();

  // --- Article Stream State (Clean Svelte 5 Runes) ---
  let extraArticles = $state<NewsArticle[]>([]);
  let extraQueue = $state<NewsArticle[]>([]);
  let activeArticle = $state<NewsArticle | null>(null);
  let isLoadingNext = $state(false);
  let hasMore = $state(true);
  let sentinelEl = $state<HTMLElement | null>(null);

  let initialArticle = $derived(data.article);
  let currentArticle = $derived(activeArticle || initialArticle || null);
  let articles = $derived<NewsArticle[]>([
    ...(initialArticle ? [initialArticle] : []),
    ...extraArticles.filter(
      (item, idx, self) =>
        item.id !== initialArticle?.id &&
        item.slug !== initialArticle?.slug &&
        self.findIndex((s) => s.id === item.id || s.slug === item.slug) === idx
    ),
  ]);

  let popularList = $derived<NewsArticle[]>(data.popularNews || data.related || []);

  // Sync state if navigation changes the active article
  $effect(() => {
    if (data?.article) {
      extraArticles = [];
      extraQueue = data.related ? [...data.related] : [];
      activeArticle = null;
      hasMore = true;
    }
  });

  // --- Font Size Adjustment ---
  let fontScale = $state<'normal' | 'lg' | 'xl'>('normal');

  function changeFontSize(delta: number) {
    if (delta > 0) {
      if (fontScale === 'normal') fontScale = 'lg';
      else if (fontScale === 'lg') fontScale = 'xl';
    } else {
      if (fontScale === 'xl') fontScale = 'lg';
      else if (fontScale === 'lg') fontScale = 'normal';
    }
  }

  // --- Text-to-Speech (TTS) Narration ---
  let isSpeaking = $state(false);
  let isPaused = $state(false);
  let speechRate = $state(1.0);
  let speechSupported = $state(false);
  let speakingArticleId = $state<string | null>(null);

  function getCleanArticleText(art: NewsArticle): string {
    if (!art) return '';
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = art.body || '';
    const bodyText = tempDiv.textContent || tempDiv.innerText || '';
    return `${art.title}. ${art.excerpt || ''}. ${bodyText}`;
  }

  // Chrome/Edge return an empty voice list until the async 'voiceschanged'
  // event fires at least once — without this, picking a Turkish voice below
  // silently fails on first use and the browser falls back to its default
  // (usually English) voice even though utterance.lang is set correctly.
  let voicesReady = $state(false);
  function loadVoices() {
    if (typeof window === 'undefined' || !('speechSynthesis' in window)) return;
    if (window.speechSynthesis.getVoices().length > 0) voicesReady = true;
  }

  function pickVoice(lang: string): SpeechSynthesisVoice | null {
    const voices = window.speechSynthesis.getVoices();
    const prefix = lang.split('-')[0];
    return voices.find((v) => v.lang === lang) || voices.find((v) => v.lang.startsWith(prefix)) || null;
  }

  function toggleSpeech(art: NewsArticle) {
    if (!speechSupported || !art) return;
    const synth = window.speechSynthesis;

    if (isSpeaking && speakingArticleId === art.id) {
      if (isPaused) {
        synth.resume();
        isPaused = false;
      } else {
        synth.pause();
        isPaused = true;
      }
    } else {
      synth.cancel();
      const text = getCleanArticleText(art);
      const lang = art.locale === 'en' ? 'en-US' : 'tr-TR';
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = lang;
      const voice = pickVoice(lang);
      if (voice) utterance.voice = voice;
      utterance.rate = speechRate;

      utterance.onend = () => {
        isSpeaking = false;
        isPaused = false;
        speakingArticleId = null;
      };

      utterance.onerror = () => {
        isSpeaking = false;
        isPaused = false;
        speakingArticleId = null;
      };

      synth.speak(utterance);
      isSpeaking = true;
      isPaused = false;
      speakingArticleId = art.id;
    }
  }

  function stopSpeech() {
    if (!speechSupported) return;
    window.speechSynthesis.cancel();
    isSpeaking = false;
    isPaused = false;
    speakingArticleId = null;
  }

  function changeRate(newRate: number) {
    speechRate = newRate;
    if (isSpeaking && activeArticle) {
      const current = activeArticle;
      stopSpeech();
      toggleSpeech(current);
    }
  }

  // --- Reactions / Emojis ---
  let userReactions = $state<Record<string, string>>({});
  let reactionStats = $state<Record<string, Record<string, number>>>({});

  const DEFAULT_REACTIONS: Record<string, number> = {
    like: 42,
    claps: 18,
    wow: 9,
    sad: 3,
    angry: 2,
  };

  function getReactions(articleId: string): Record<string, number> {
    const key = articleId || 'default';
    return reactionStats[key] || DEFAULT_REACTIONS;
  }

  function react(articleId: string, type: string) {
    const key = articleId || 'default';
    const current = userReactions[key];
    const currentStats = reactionStats[key] || { ...DEFAULT_REACTIONS };
    const nextStats = { ...currentStats };

    if (current === type) {
      // Clicking the active reaction again removes it.
      nextStats[type] = Math.max(0, (nextStats[type] || 1) - 1);
      const { [key]: _removed, ...rest } = userReactions;
      userReactions = rest;
      if (typeof localStorage !== 'undefined') {
        localStorage.removeItem(`reaction_${key}`);
      }
    } else {
      if (current) {
        nextStats[current] = Math.max(0, (nextStats[current] || 1) - 1);
      }
      nextStats[type] = (nextStats[type] || 0) + 1;
      userReactions = { ...userReactions, [key]: type };
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(`reaction_${key}`, type);
      }
    }

    reactionStats = { ...reactionStats, [key]: nextStats };
  }

  function getReadingTime(art?: NewsArticle): number {
    if (!art?.body) return 2;
    const words = art.body.replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
    return Math.max(1, Math.ceil(words / 180));
  }

  // --- Infinite Scroll Loader ---
  async function loadNextArticle() {
    if (isLoadingNext || !hasMore) return;
    isLoadingNext = true;

    try {
      let nextArt: NewsArticle | null = null;

      // 1. Check queue from related articles
      while (extraQueue.length > 0) {
        const candidate = extraQueue.shift()!;
        if (!articles.some((a) => a.slug === candidate.slug)) {
          if (!candidate.body || candidate.body.length < 50) {
            const full = await fetchApi<NewsArticle>(`/api/v1/news/tr/${candidate.slug}`);
            nextArt = full || candidate;
          } else {
            nextArt = candidate;
          }
          break;
        }
      }

      // 2. If queue is empty, fetch additional articles from API
      if (!nextArt) {
        const res = await fetchApi<NewsArticle[]>(`/api/v1/news?locale=tr&limit=20`);
        if (res && res.length > 0) {
          const unread = res.filter((a) => !articles.some((existing) => existing.slug === a.slug));
          if (unread.length > 0) {
            nextArt = unread[0];
            extraQueue.push(...unread.slice(1));
          } else {
            hasMore = false;
          }
        } else {
          hasMore = false;
        }
      }

      if (nextArt && nextArt.slug) {
        extraArticles = [...extraArticles, nextArt];
        await tick();
        observeArticles();
      }
    } catch (err) {
      console.error('Failed to load next article:', err);
    } finally {
      isLoadingNext = false;
    }
  }

  let articleObserver: IntersectionObserver | null = null;
  let sentinelObserver: IntersectionObserver | null = null;

  function observeArticles() {
    if (typeof window === 'undefined' || !('IntersectionObserver' in window)) return;

    if (articleObserver) articleObserver.disconnect();

    articleObserver = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            const slug = entry.target.getAttribute('data-slug');
            const targetArticle = articles.find((a) => a.slug === slug);
            if (targetArticle && activeArticle?.slug !== targetArticle.slug) {
              activeArticle = targetArticle;
              const newUrl = `/${slugify(targetArticle.category)}/${targetArticle.slug}`;
              try {
                if (typeof window !== 'undefined' && window.history?.replaceState) {
                  window.history.replaceState(null, '', newUrl);
                }
              } catch {
                // Ignore history update error
              }
              document.title = `${targetArticle.seo_title || targetArticle.title} — HydroPost`;
            }
          }
        }
      },
      { rootMargin: '-20% 0px -60% 0px', threshold: 0 }
    );

    document.querySelectorAll('[data-article-block]').forEach((el) => {
      articleObserver?.observe(el);
    });
  }

  onMount(() => {
    speechSupported = typeof window !== 'undefined' && 'speechSynthesis' in window;
    if (speechSupported) {
      loadVoices();
      window.speechSynthesis.addEventListener('voiceschanged', loadVoices);
    }

    if (typeof localStorage !== 'undefined' && data?.article?.id) {
      const saved = localStorage.getItem(`reaction_${data.article.id}`);
      if (saved) userReactions[data.article.id] = saved;
    }

    if (sentinelEl) {
      sentinelObserver = new IntersectionObserver(
        (entries) => {
          if (entries[0].isIntersecting) {
            loadNextArticle();
          }
        },
        { rootMargin: '300px' }
      );
      sentinelObserver.observe(sentinelEl);
    }

    observeArticles();

    return () => {
      if (articleObserver) articleObserver.disconnect();
      if (sentinelObserver) sentinelObserver.disconnect();
      if (speechSupported) {
        window.speechSynthesis.cancel();
        window.speechSynthesis.removeEventListener('voiceschanged', loadVoices);
      }
    };
  });
</script>

<svelte:head>
  <title>{activeArticle?.seo_title || activeArticle?.title || data?.article?.title || 'Haber Detay'} — HydroPost</title>
  <meta name="description" content={activeArticle?.seo_description || activeArticle?.excerpt || data?.article?.excerpt || ''} />
  <!-- OpenGraph -->
  <meta property="og:title" content={activeArticle?.title || data?.article?.title || ''} />
  <meta property="og:description" content={activeArticle?.excerpt || data?.article?.excerpt || ''} />
  <meta property="og:type" content="article" />
  <meta property="og:image" content={getMediaUrl(activeArticle?.cover_media_id || data?.article?.cover_media_id, activeArticle?.category || data?.article?.category)} />
  {#if data?.jsonLd}
    <!-- Google News JSON-LD -->
    {@html `<script type="application/ld+json">${JSON.stringify(data.jsonLd)}<\/script>`}
  {/if}
</svelte:head>

<div class="py-4">
  <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
    <!-- Main Left Column: Article Stream & Related News -->
    <div class="lg:col-span-8 min-w-0">
      {#each articles as art, index (`${art.id || art.slug || index}_${index}`)}
        {@const badge = getCategoryBadge(art.category)}
        {@const readingTime = getReadingTime(art)}
        {@const reactions = getReactions(art.id)}

        <!-- Transition divider between infinite scroll articles -->
        {#if index > 0}
          <div class="my-14 border-t-2 border-dashed border-[var(--hp-border)]"></div>
        {/if}

        <div
          data-article-block
          data-slug={art.slug}
          data-category={art.category}
          data-title={art.title}
          class="article-container mb-12"
        >
          <!-- Article Header -->
          <header class="space-y-4 mb-6">
            <div class="flex items-center justify-between flex-wrap gap-2">
              {#if art.category}
                <a
                  href={`/${slugify(art.category)}`}
                  class={`inline-block px-3 py-1 rounded ${badge.bg} ${badge.text} text-xs font-bold uppercase tracking-wider shadow`}
                >
                  {art.category}
                </a>
              {/if}
              <div class="flex items-center gap-3 text-xs text-[var(--hp-muted)]">
                <span>⏱ {readingTime} dakika okuma</span>
                <!-- Font sizing controls -->
                <div class="flex items-center bg-[var(--hp-surface)] border border-[var(--hp-border)] rounded-lg p-0.5">
                  <button
                    type="button"
                    onclick={() => changeFontSize(-1)}
                    class="px-2 py-0.5 hover:text-[var(--hp-ink)] rounded transition-colors text-xs font-bold cursor-pointer"
                    title="Yazıyı Küçült"
                  >
                    A-
                  </button>
                  <span class="text-[var(--hp-subtle)]">|</span>
                  <button
                    type="button"
                    onclick={() => changeFontSize(1)}
                    class="px-2 py-0.5 hover:text-[var(--hp-ink)] rounded transition-colors text-xs font-bold cursor-pointer"
                    title="Yazıyı Büyüt"
                  >
                    A+
                  </button>
                </div>
              </div>
            </div>

            <h1 class="text-2xl sm:text-3xl md:text-5xl font-extrabold text-[var(--hp-ink)] leading-tight tracking-tight">
              {art.title}
            </h1>

            {#if art.excerpt}
              <p class="text-base sm:text-lg text-[var(--hp-ink-soft)] font-medium leading-relaxed border-l-4 border-[var(--hp-primary)] pl-4 py-2 bg-[var(--hp-surface-sunken)] rounded-r-xl">
                {art.excerpt}
              </p>
            {/if}

            <!-- Audio Player Bar (Text to Speech) -->
            {#if speechSupported}
              <div class="flex flex-wrap items-center justify-between gap-3 p-4 rounded-2xl bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)]">
                <div class="flex items-center gap-3">
                  <button
                    type="button"
                    onclick={() => toggleSpeech(art)}
                    class="w-10 h-10 rounded-full bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white flex items-center justify-center font-bold text-base cursor-pointer transition-colors"
                    aria-label="Haberi Dinle"
                  >
                    {#if isSpeaking && speakingArticleId === art.id && !isPaused}
                      ⏸
                    {:else}
                      ▶
                    {/if}
                  </button>
                  <div>
                    <div class="text-xs font-bold text-[var(--hp-ink)]">
                      {isSpeaking && speakingArticleId === art.id ? (isPaused ? 'Sesli Dinleme Duraklatıldı' : 'Sesli Dinleniyor...') : 'Bu Haberi Sesli Dinle'}
                    </div>
                    <div class="text-[11px] text-[var(--hp-muted)]">Doğal tarayıcı seslendiricisi ile kesintisiz haber akışı</div>
                  </div>
                </div>

                <div class="flex items-center gap-2">
                  {#if isSpeaking && speakingArticleId === art.id}
                    <button
                      type="button"
                      onclick={stopSpeech}
                      class="px-2.5 py-1 rounded bg-[var(--hp-border)] hover:bg-[var(--hp-border-strong)] text-[var(--hp-ink-soft)] text-xs cursor-pointer font-medium"
                    >
                      Durdur
                    </button>
                  {/if}
                  <div class="flex items-center gap-1 text-[11px] bg-[var(--hp-surface)] px-2.5 py-1 rounded-lg border border-[var(--hp-border)] text-[var(--hp-muted)]">
                    <span>Hız:</span>
                    {#each [1.0, 1.25, 1.5] as rate}
                      <button
                        type="button"
                        onclick={() => changeRate(rate)}
                        class={`px-1.5 py-0.5 rounded ${speechRate === rate ? 'bg-[var(--hp-primary)] text-white font-bold' : 'hover:text-[var(--hp-ink)]'}`}
                      >
                        {rate}x
                      </button>
                    {/each}
                  </div>
                </div>
              </div>
            {/if}

            <div class="flex items-center justify-between py-3 border-y border-[var(--hp-border)] text-xs text-[var(--hp-muted)]">
              <div class="flex items-center gap-3 flex-wrap">
                {#if art.author_name}
                  <span class="font-bold text-[var(--hp-ink)] flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full bg-[var(--hp-primary)]"></span>
                    {art.author_name}
                  </span>
                {/if}
                <span class="px-2.5 py-0.5 rounded-full bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] text-[var(--hp-ink-soft)] font-mono text-[11px]">
                  Mahreç: {art.author_name ? 'Özel Haber' : 'Haber Merkezi'}
                </span>
                <span>Yayınlanma: {new Date(art.created_at).toLocaleDateString('tr-TR', { day: 'numeric', month: 'long', year: 'numeric' })}</span>
              </div>

              <div class="flex items-center gap-2">
                <button
                  type="button"
                  onclick={() => {
                    if (navigator.share) {
                      navigator.share({ title: art.title, url: window.location.href });
                    } else {
                      navigator.clipboard.writeText(window.location.href);
                      alert('Haber bağlantısı kopyalandı!');
                    }
                  }}
                  class="px-3 py-1.5 rounded-lg bg-[var(--hp-surface-sunken)] hover:bg-[var(--hp-border)] text-[var(--hp-ink-soft)] font-semibold transition-colors cursor-pointer"
                >
                  Paylaş ↗
                </button>
              </div>
            </div>
          </header>

          <!-- Cover Image -->
          <div class="rounded-2xl overflow-hidden bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] mb-8 aspect-video relative">
            <img
              src={getMediaUrl(art.cover_media_id, art.category, art.slug)}
              alt={art.title}
              class="w-full h-full object-cover"
            />
          </div>

          <!-- In-Article Ad Slot -->
          <AdSlot ad={data.inArticleAd} zone="haber_ici" />

          <!-- Article Body (Quill Delta JSON is automatically transformed to clean HTML) -->
          <!-- Tailwind Typography sets explicit font sizes on nested p/li/etc,
               so a generic text-xl on the container doesn't cascade to them —
               use the plugin's own prose-lg/prose-xl size modifiers instead. -->
          <article
            class={`prose max-w-none text-[var(--hp-ink-soft)] transition-all duration-150 ${
              fontScale === 'xl' ? 'prose-xl' : fontScale === 'lg' ? 'prose-lg' : 'prose-base'
            }`}
          >
            {@html sanitizeHtml(art.body)}
          </article>

          <!-- Tags -->
          {#if art.tags}
            {@const tagsList = art.tags.split(',').map((t: string) => t.trim()).filter(Boolean)}
            {#if tagsList.length > 0}
              <div class="mt-8 pt-6 border-t border-[var(--hp-border)] flex flex-wrap items-center gap-2">
                <span class="text-xs text-[var(--hp-muted)] font-medium">Etiketler:</span>
                {#each tagsList as tag}
                  <span class="px-2.5 py-1 rounded-lg bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] text-xs font-medium text-[var(--hp-ink-soft)] hover:border-[var(--hp-border-strong)] transition-colors">
                    #{tag}
                  </span>
                {/each}
              </div>
            {/if}
          {/if}

          <!-- Article Reactions / Reader Feedback -->
          <div class="mt-10 p-5 rounded-2xl bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] text-center space-y-3">
            <h3 class="text-sm font-bold text-[var(--hp-ink-soft)] uppercase tracking-wider">Bu habere tepkiniz nedir?</h3>
            <div class="flex items-center justify-center gap-3 sm:gap-6 flex-wrap">
              {#each [
                { key: 'like', emoji: '👍', label: 'Beğendim' },
                { key: 'claps', emoji: '👏', label: 'Tebrikler' },
                { key: 'wow', emoji: '😮', label: 'Şaşırdım' },
                { key: 'sad', emoji: '😢', label: 'Üzüldüm' },
                { key: 'angry', emoji: '😡', label: 'Kızdım' },
              ] as r}
                <button
                  type="button"
                  onclick={() => react(art.id, r.key)}
                  class={`flex flex-col items-center gap-1 px-3 py-2 rounded-xl border transition-colors cursor-pointer ${
                    userReactions[art.id || 'default'] === r.key
                      ? 'bg-[var(--hp-primary-light)] border-[var(--hp-primary)] text-[var(--hp-primary-dark)]'
                      : 'bg-[var(--hp-surface)] border-[var(--hp-border)] text-[var(--hp-ink-soft)] hover:border-[var(--hp-border-strong)]'
                  }`}
                >
                  <span class="text-2xl">{r.emoji}</span>
                  <span class="text-[11px] font-semibold">{r.label}</span>
                  <span class="text-[10px] text-[var(--hp-muted)] font-bold">{reactions[r.key] || 0}</span>
                </button>
              {/each}
            </div>
          </div>

          {#if data.commentsEnabled && art.slug === data.article?.slug}
            <CommentsSection slug={art.slug} comments={data.comments} requireEmail={data.commentsRequireEmail} />
          {/if}

          <!-- Bottom Ad Slot -->
          <AdSlot ad={data.footerAd} zone="footer" class="my-8" />
        </div>
      {/each}

      <!-- Infinite Scroll Sentinel & Loader -->
      <div bind:this={sentinelEl} class="py-8 text-center">
        {#if isLoadingNext}
          <div class="inline-flex items-center gap-3 px-5 py-2.5 rounded-full bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] text-[var(--hp-ink-soft)] text-xs font-semibold">
            <div class="w-4 h-4 border-2 border-[var(--hp-primary)] border-t-transparent rounded-full animate-spin"></div>
            <span>Sıradaki haber yükleniyor...</span>
          </div>
        {:else if hasMore}
          <div class="text-xs text-[var(--hp-muted)] flex items-center justify-center gap-2">
            <span>↓ Aşağı kaydırarak sonraki habere geçin</span>
          </div>
        {:else}
          <div class="text-xs text-[var(--hp-muted)] py-4">
            ✓ Akıştaki tüm haberleri okudunuz
          </div>
        {/if}
      </div>

      <!-- Related News Grid at Bottom -->
      {#if data?.related && data.related.length > 0}
        <section class="mt-8 pt-8 border-t border-[var(--hp-border)]">
          <h3 class="text-lg md:text-xl font-bold text-[var(--hp-ink)] mb-5">
            Kategorideki Diğer Başlıklar
          </h3>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {#each data.related as r, i (r.id)}
              <NewsCard item={r} variant="tile" index={i} />
            {/each}
          </div>
        </section>
      {/if}
    </div>

    <!-- Right Sticky Column: Popüler Haberler & Sidebar Ad -->
    <aside class="hidden lg:block lg:col-span-4 sticky top-28 space-y-6">
      <div class="bg-[var(--hp-surface)] border border-[var(--hp-border)] rounded-2xl p-5 sm:p-6">
        <h3 class="text-base font-bold text-[var(--hp-ink)] mb-5">
          Popüler Haberler
        </h3>

        <div class="space-y-5">
          {#each popularList.slice(0, 6) as item, idx (item.id)}
            <NewsCard {item} variant="row" index={idx} />
          {/each}
        </div>
      </div>

      <!-- Sticky Sidebar Ad Slot -->
      <AdSlot ad={data.sidebarAd} zone="sidebar" />
    </aside>
  </div>
</div>

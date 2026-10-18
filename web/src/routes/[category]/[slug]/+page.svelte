<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, sanitizeHtml, getCategoryBadge, fetchApi, slugify, type NewsArticle } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';
  import { onMount, tick } from 'svelte';
  import { replaceState } from '$app/navigation';

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
    ...extraArticles,
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
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = art.locale === 'en' ? 'en-US' : 'tr-TR';
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
    if (current === type) return;

    const currentStats = reactionStats[key] || { ...DEFAULT_REACTIONS };
    const nextStats = { ...currentStats };
    if (current) {
      nextStats[current] = Math.max(0, (nextStats[current] || 1) - 1);
    }
    nextStats[type] = (nextStats[type] || 0) + 1;

    userReactions = { ...userReactions, [key]: type };
    reactionStats = { ...reactionStats, [key]: nextStats };

    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(`reaction_${key}`, type);
    }
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
                replaceState(newUrl, {});
              } catch {
                // Ignore router conflict
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

<div class="max-w-7xl mx-auto py-4 px-2 sm:px-0">
  <div class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
    <!-- Main Left Column: Article Stream & Related News -->
    <div class="lg:col-span-8 min-w-0">
      {#each articles as art, index (art.id || art.slug)}
        {@const badge = getCategoryBadge(art.category)}
        {@const readingTime = getReadingTime(art)}
        {@const reactions = getReactions(art.id)}

        <!-- Transition divider between infinite scroll articles -->
        {#if index > 0}
          <div class="relative my-14 py-6 border-t-2 border-dashed border-zinc-200 text-center">
            <div class="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-zinc-100 border border-zinc-300 text-xs font-bold text-zinc-700 shadow-xs uppercase tracking-wider">
              <span class="w-2 h-2 rounded-full bg-red-600 animate-pulse"></span>
              Sıradaki Haber: {art.category || 'Gündem'}
            </div>
          </div>
        {/if}

        <div
          data-article-block
          data-slug={art.slug}
          data-category={art.category}
          data-title={art.title}
          class="article-container mb-12"
        >
          <!-- Breadcrumbs -->
          <nav class="flex items-center gap-2 text-xs text-zinc-500 mb-4 select-none">
            <a href="/" class="hover:text-zinc-700 transition-colors">Ana Sayfa</a>
            <span>/</span>
            <a href={`/${slugify(art.category)}`} class="hover:text-zinc-700 transition-colors uppercase font-semibold">
              {art.category || 'Haber'}
            </a>
            <span>/</span>
            <span class="text-zinc-800 truncate max-w-xs">{art.title}</span>
          </nav>

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
              <div class="flex items-center gap-3 text-xs text-zinc-500">
                <span>⏱ {readingTime} dakika okuma</span>
                <!-- Font sizing controls -->
                <div class="flex items-center bg-white border border-zinc-200 rounded-lg p-0.5 shadow-2xs">
                  <button
                    type="button"
                    onclick={() => changeFontSize(-1)}
                    class="px-2 py-0.5 hover:text-zinc-900 rounded transition-colors text-xs font-bold cursor-pointer"
                    title="Yazıyı Küçült"
                  >
                    A-
                  </button>
                  <span class="text-zinc-300">|</span>
                  <button
                    type="button"
                    onclick={() => changeFontSize(1)}
                    class="px-2 py-0.5 hover:text-zinc-900 rounded transition-colors text-xs font-bold cursor-pointer"
                    title="Yazıyı Büyüt"
                  >
                    A+
                  </button>
                </div>
              </div>
            </div>

            <h1 class="text-2xl sm:text-3xl md:text-5xl font-extrabold text-zinc-900 leading-tight tracking-tight">
              {art.title}
            </h1>

            {#if art.excerpt}
              <p class="text-base sm:text-lg text-zinc-700 font-medium leading-relaxed border-l-4 border-red-600 pl-4 py-2 bg-zinc-50 rounded-r-xl">
                {art.excerpt}
              </p>
            {/if}

            <!-- Audio Player Bar (Text to Speech) -->
            {#if speechSupported}
              <div class="flex flex-wrap items-center justify-between gap-3 p-4 rounded-2xl bg-zinc-50 border border-zinc-200 shadow-xs">
                <div class="flex items-center gap-3">
                  <button
                    type="button"
                    onclick={() => toggleSpeech(art)}
                    class="w-10 h-10 rounded-full bg-red-600 hover:bg-red-700 text-white flex items-center justify-center font-bold text-base shadow-sm cursor-pointer transition-colors"
                    aria-label="Haberi Dinle"
                  >
                    {#if isSpeaking && speakingArticleId === art.id && !isPaused}
                      ⏸
                    {:else}
                      ▶
                    {/if}
                  </button>
                  <div>
                    <div class="text-xs font-bold text-zinc-900">
                      {isSpeaking && speakingArticleId === art.id ? (isPaused ? 'Sesli Dinleme Duraklatıldı' : 'Sesli Dinleniyor...') : 'Bu Haberi Sesli Dinle'}
                    </div>
                    <div class="text-[11px] text-zinc-500">Doğal tarayıcı seslendiricisi ile kesintisiz haber akışı</div>
                  </div>
                </div>

                <div class="flex items-center gap-2">
                  {#if isSpeaking && speakingArticleId === art.id}
                    <button
                      type="button"
                      onclick={stopSpeech}
                      class="px-2.5 py-1 rounded bg-zinc-200 hover:bg-zinc-300 text-zinc-700 text-xs cursor-pointer font-medium"
                    >
                      Durdur
                    </button>
                  {/if}
                  <div class="flex items-center gap-1 text-[11px] bg-white px-2.5 py-1 rounded-lg border border-zinc-200 text-zinc-600 shadow-2xs">
                    <span>Hız:</span>
                    {#each [1.0, 1.25, 1.5] as rate}
                      <button
                        type="button"
                        onclick={() => changeRate(rate)}
                        class={`px-1.5 py-0.5 rounded ${speechRate === rate ? 'bg-red-600 text-white font-bold' : 'hover:text-zinc-900'}`}
                      >
                        {rate}x
                      </button>
                    {/each}
                  </div>
                </div>
              </div>
            {/if}

            <div class="flex items-center justify-between py-3 border-y border-zinc-200 text-xs text-zinc-500">
              <div class="flex items-center gap-3 flex-wrap">
                {#if art.author_name}
                  <span class="font-bold text-zinc-900 flex items-center gap-1.5">
                    <span class="w-2 h-2 rounded-full bg-red-600"></span>
                    {art.author_name}
                  </span>
                {/if}
                <span class="px-2.5 py-0.5 rounded-full bg-zinc-100 border border-zinc-200 text-zinc-700 font-mono text-[11px]">
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
                  class="px-3 py-1.5 rounded-lg bg-zinc-100 hover:bg-zinc-200 text-zinc-700 font-semibold transition-colors cursor-pointer"
                >
                  Paylaş ↗
                </button>
              </div>
            </div>
          </header>

          <!-- Cover Image -->
          <div class="rounded-2xl overflow-hidden bg-zinc-100 border border-zinc-200 mb-8 aspect-video shadow-xs relative">
            <img
              src={getMediaUrl(art.cover_media_id, art.category)}
              alt={art.title}
              class="w-full h-full object-cover"
            />
          </div>

          <!-- In-Article Ad Slot -->
          <AdSlot zone="haber_ici" />

          <!-- Article Body (Quill Delta JSON is automatically transformed to clean HTML) -->
          <article
            class={`prose max-w-none text-zinc-800 leading-relaxed transition-all duration-150 ${
              fontScale === 'xl' ? 'text-2xl leading-loose' : fontScale === 'lg' ? 'text-xl leading-relaxed' : 'sm:text-lg'
            }`}
          >
            {@html sanitizeHtml(art.body)}
          </article>

          <!-- Tags -->
          {#if art.tags}
            {@const tagsList = art.tags.split(',').map((t: string) => t.trim()).filter(Boolean)}
            {#if tagsList.length > 0}
              <div class="mt-8 pt-6 border-t border-zinc-200 flex flex-wrap items-center gap-2">
                <span class="text-xs text-zinc-500 font-medium">Etiketler:</span>
                {#each tagsList as tag}
                  <span class="px-2.5 py-1 rounded-lg bg-zinc-100 border border-zinc-200 text-xs font-medium text-zinc-700 hover:border-zinc-300 transition-colors">
                    #{tag}
                  </span>
                {/each}
              </div>
            {/if}
          {/if}

          <!-- Article Reactions / Reader Feedback -->
          <div class="mt-10 p-5 rounded-2xl bg-zinc-50 border border-zinc-200 text-center space-y-3">
            <h3 class="text-sm font-bold text-zinc-800 uppercase tracking-wider">Bu habere tepkiniz nedir?</h3>
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
                      ? 'bg-red-50 border-red-500 text-red-900 shadow-xs'
                      : 'bg-white border-zinc-200 text-zinc-700 hover:border-zinc-300'
                  }`}
                >
                  <span class="text-2xl">{r.emoji}</span>
                  <span class="text-[11px] font-semibold">{r.label}</span>
                  <span class="text-[10px] text-zinc-500 font-bold">{reactions[r.key] || 0}</span>
                </button>
              {/each}
            </div>
          </div>

          <!-- Bottom Ad Slot -->
          <AdSlot zone="footer" class="my-8" />
        </div>
      {/each}

      <!-- Infinite Scroll Sentinel & Loader -->
      <div bind:this={sentinelEl} class="py-8 text-center">
        {#if isLoadingNext}
          <div class="inline-flex items-center gap-3 px-5 py-2.5 rounded-full bg-zinc-100 border border-zinc-200 text-zinc-700 text-xs font-semibold shadow-xs">
            <div class="w-4 h-4 border-2 border-red-600 border-t-transparent rounded-full animate-spin"></div>
            <span>Sıradaki haber yükleniyor...</span>
          </div>
        {:else if hasMore}
          <div class="text-xs text-zinc-500 flex items-center justify-center gap-2">
            <span>↓ Aşağı kaydırarak sonraki habere geçin</span>
          </div>
        {:else}
          <div class="text-xs text-zinc-500 py-4">
            ✓ Akıştaki tüm haberleri okudunuz
          </div>
        {/if}
      </div>

      <!-- Related News Grid at Bottom -->
      {#if data?.related && data.related.length > 0}
        <section class="mt-8 pt-8 border-t border-zinc-200">
          <h3 class="text-lg font-bold text-zinc-900 mb-4 flex items-center gap-2">
            <span class="w-2.5 h-2.5 rounded bg-red-600"></span>
            Kategorideki Diğer Başlıklar
          </h3>
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
            {#each data.related as r}
              <a
                href={`/${slugify(r.category)}/${r.slug}`}
                class="group block p-3.5 rounded-2xl bg-white border border-zinc-200 shadow-xs cursor-pointer"
              >
                <div class="aspect-video rounded-xl overflow-hidden bg-zinc-100 mb-2.5 relative">
                  <img
                    src={getMediaUrl(r.cover_media_id, r.category)}
                    alt={r.title}
                    class="w-full h-full object-cover"
                    loading="lazy"
                  />
                </div>
                <h4 class="text-xs font-bold text-zinc-900 line-clamp-2 leading-snug group-hover:underline">
                  {r.title}
                </h4>
              </a>
            {/each}
          </div>
        </section>
      {/if}
    </div>

    <!-- Right Sticky Column: Popüler Haberler & Sidebar Ad -->
    <aside class="hidden lg:block lg:col-span-4 sticky top-6 space-y-6">
      <div class="bg-zinc-50 border border-zinc-200 rounded-2xl p-5 shadow-xs">
        <h3 class="text-base font-extrabold text-zinc-900 uppercase tracking-wider flex items-center gap-2 border-b border-zinc-200 pb-3 mb-3">
          <span class="w-2.5 h-2.5 rounded bg-red-600"></span>
          Popüler Haberler
        </h3>

        <div class="divide-y divide-zinc-200">
          {#each popularList.slice(0, 6) as item, idx}
            <a
              href={`/${slugify(item.category)}/${item.slug}`}
              class="group flex items-start gap-3 py-3 first:pt-1 last:pb-1 cursor-pointer"
            >
              <span
                class={`w-6 h-6 rounded-md flex items-center justify-center font-mono font-bold text-xs shrink-0 ${
                  idx === 0 ? 'bg-red-600 text-white shadow-2xs' : 'bg-zinc-200 text-zinc-700'
                }`}
              >
                0{idx + 1}
              </span>

              <div class="w-16 h-12 rounded-lg overflow-hidden bg-zinc-200 shrink-0 relative">
                <img
                  src={getMediaUrl(item.cover_media_id, item.category)}
                  alt={item.title}
                  class="w-full h-full object-cover"
                  loading="lazy"
                />
              </div>

              <div class="flex-1 min-w-0 space-y-1">
                {#if item.category}
                  <span class="text-[10px] font-bold text-red-600 uppercase tracking-wider block">
                    {item.category}
                  </span>
                {/if}
                <h4 class="text-xs font-bold text-zinc-900 line-clamp-2 leading-snug group-hover:underline">
                  {item.title}
                </h4>
              </div>
            </a>
          {/each}
        </div>
      </div>

      <!-- Sticky Sidebar Ad Slot -->
      <AdSlot zone="sidebar" />
    </aside>
  </div>
</div>

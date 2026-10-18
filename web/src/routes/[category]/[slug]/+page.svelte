<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, sanitizeHtml, getCategoryBadge, fetchApi, type NewsArticle } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';
  import { onMount, tick } from 'svelte';

  let { data }: PageProps = $props();

  // --- Article Stream State (Infinite Scroll) ---
  let articles = $state<NewsArticle[]>([]);
  let queue = $state<NewsArticle[]>([]);
  let activeArticle = $state<NewsArticle | null>(null);
  let isLoadingNext = $state(false);
  let hasMore = $state(true);
  let sentinelEl = $state<HTMLElement | null>(null);

  // Sync state whenever page load data changes
  $effect(() => {
    articles = [data.article];
    queue = [...(data.related || [])];
    activeArticle = data.article;
    hasMore = queue.length > 0 || true;
  });

  // --- Reading Progress State ---
  let scrollProgress = $state(0);
  let showStickyHeader = $state(false);

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
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = art.body || '';
    const bodyText = tempDiv.textContent || tempDiv.innerText || '';
    return `${art.title}. ${art.excerpt || ''}. ${bodyText}`;
  }

  function toggleSpeech(art: NewsArticle) {
    if (!speechSupported) return;
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

  function getReactions(articleId: string) {
    if (!reactionStats[articleId]) {
      reactionStats[articleId] = {
        like: 42,
        claps: 18,
        wow: 9,
        sad: 3,
        angry: 2,
      };
    }
    return reactionStats[articleId];
  }

  function react(articleId: string, type: string) {
    const current = userReactions[articleId];
    if (current === type) return;

    const stats = getReactions(articleId);
    if (current) {
      stats[current] = Math.max(0, (stats[current] || 1) - 1);
    }
    userReactions[articleId] = type;
    stats[type] = (stats[type] || 0) + 1;

    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(`reaction_${articleId}`, type);
    }
  }

  function getReadingTime(art: NewsArticle): number {
    const words = (art.body || '').replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
    return Math.max(1, Math.ceil(words / 180));
  }

  // --- Infinite Scroll Loader ---
  async function loadNextArticle() {
    if (isLoadingNext || !hasMore) return;
    isLoadingNext = true;

    try {
      let nextArt: NewsArticle | null = null;

      // 1. Check queue from related articles
      while (queue.length > 0) {
        const candidate = queue.shift()!;
        if (!articles.some((a) => a.slug === candidate.slug)) {
          // If body is missing or short summary, fetch full article detail
          if (!candidate.body || candidate.body.length < 50) {
            const full = await fetchApi<NewsArticle>(`/api/v1/news/tr/${candidate.slug}`);
            nextArt = full || candidate;
          } else {
            nextArt = candidate;
          }
          break;
        }
      }

      // 2. If queue is exhausted, fetch more from API
      if (!nextArt) {
        const currentCat = activeArticle?.category || 'gundem';
        const res = await fetchApi<NewsArticle[]>(`/api/v1/news?locale=tr&limit=20`);
        if (res && res.length > 0) {
          const unread = res.filter((a) => !articles.some((existing) => existing.slug === a.slug));
          if (unread.length > 0) {
            nextArt = unread[0];
            // Push remaining unread to queue
            queue.push(...unread.slice(1));
          } else {
            hasMore = false;
          }
        } else {
          hasMore = false;
        }
      }

      if (nextArt) {
        articles.push(nextArt);
        await tick();
        observeArticles();
      }
    } catch (err) {
      console.error('Failed to load next article:', err);
    } finally {
      isLoadingNext = false;
    }
  }

  // Observe active article for URL and Title synchronization
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
              const newUrl = `/${targetArticle.category || 'haber'}/${targetArticle.slug}`;
              window.history.replaceState(null, '', newUrl);
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

    // Load initial reaction
    if (typeof localStorage !== 'undefined' && data.article?.id) {
      const saved = localStorage.getItem(`reaction_${data.article.id}`);
      if (saved) userReactions[data.article.id] = saved;
    }

    const handleScroll = () => {
      const el = document.documentElement;
      const totalHeight = el.scrollHeight - el.clientHeight;
      if (totalHeight > 0) {
        scrollProgress = Math.min(100, Math.max(0, (window.scrollY / totalHeight) * 100));
      }
      showStickyHeader = window.scrollY > 350;
    };

    window.addEventListener('scroll', handleScroll, { passive: true });

    // Setup intersection observer for infinite scroll sentinel
    if (sentinelEl) {
      sentinelObserver = new IntersectionObserver(
        (entries) => {
          if (entries[0].isIntersecting) {
            loadNextArticle();
          }
        },
        { rootMargin: '400px' }
      );
      sentinelObserver.observe(sentinelEl);
    }

    observeArticles();

    return () => {
      window.removeEventListener('scroll', handleScroll);
      if (articleObserver) articleObserver.disconnect();
      if (sentinelObserver) sentinelObserver.disconnect();
      if (speechSupported) {
        window.speechSynthesis.cancel();
      }
    };
  });
</script>

<svelte:head>
  <title>{(activeArticle || data.article).seo_title || (activeArticle || data.article).title} — HydroPost</title>
  <meta name="description" content={(activeArticle || data.article).seo_description || (activeArticle || data.article).excerpt || (activeArticle || data.article).title} />
  <!-- OpenGraph -->
  <meta property="og:title" content={(activeArticle || data.article).title} />
  <meta property="og:description" content={(activeArticle || data.article).excerpt || (activeArticle || data.article).title} />
  <meta property="og:type" content="article" />
  <meta property="og:image" content={getMediaUrl((activeArticle || data.article).cover_media_id, (activeArticle || data.article).category)} />
  <!-- Google News JSON-LD -->
  <script type="application/ld+json">
    {@html JSON.stringify(data.jsonLd)}
  </script>
</svelte:head>

<!-- Sticky Top Progress & Quick Actions Bar -->
<div class="fixed top-0 left-0 right-0 z-50 transition-all duration-300">
  <!-- Progress line -->
  <div class="h-1 bg-zinc-800 w-full">
    <div
      class="h-full bg-gradient-to-r from-red-600 via-rose-500 to-amber-500 transition-all duration-75"
      style={`width: ${scrollProgress}%`}
    ></div>
  </div>

  {#if showStickyHeader && activeArticle}
    {@const badge = getCategoryBadge(activeArticle.category)}
    <div class="bg-zinc-950/95 backdrop-blur-md border-b border-zinc-800 px-4 py-2.5 flex items-center justify-between text-xs text-zinc-300 shadow-2xl">
      <div class="flex items-center gap-3 truncate max-w-xl">
        <span class={`px-2 py-0.5 rounded ${badge.bg} ${badge.text} font-bold uppercase text-[10px] tracking-wider`}>
          {activeArticle.category || 'Haber'}
        </span>
        <span class="font-semibold truncate text-white">{activeArticle.title}</span>
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <span class="text-zinc-500 hidden sm:inline">{getReadingTime(activeArticle)} dk okuma</span>
        {#if speechSupported}
          <button
            type="button"
            onclick={() => activeArticle && toggleSpeech(activeArticle)}
            class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            {#if isSpeaking && speakingArticleId === activeArticle.id && !isPaused}
              <span>⏸</span> <span>Duraklat</span>
            {:else if isSpeaking && speakingArticleId === activeArticle.id && isPaused}
              <span>▶</span> <span>Devam</span>
            {:else}
              <span>🔊</span> <span>Dinle</span>
            {/if}
          </button>
        {/if}
        <button
          type="button"
          onclick={() => {
            if (navigator.share) {
              navigator.share({ title: activeArticle?.title, url: window.location.href });
            } else {
              navigator.clipboard.writeText(window.location.href);
              alert('Bağlantı kopyalandı!');
            }
          }}
          class="px-3 py-1 rounded bg-red-600 hover:bg-red-500 text-white font-bold cursor-pointer transition-colors shadow"
        >
          Paylaş ↗
        </button>
      </div>
    </div>
  {/if}
</div>

<div class="max-w-4xl mx-auto py-4 px-2 sm:px-0">
  <!-- Stacked Articles (Infinite Scroll Flow) -->
  {#each articles as art, index (art.id || art.slug)}
    {@const badge = getCategoryBadge(art.category)}
    {@const readingTime = getReadingTime(art)}
    {@const reactions = getReactions(art.id)}

    <!-- Transition divider between infinite scroll articles -->
    {#if index > 0}
      <div class="relative my-16 py-8 border-t-2 border-dashed border-zinc-800 text-center">
        <div class="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-zinc-900 border border-zinc-700 text-xs font-bold text-zinc-300 shadow-xl uppercase tracking-wider">
          <span class="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
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
        <a href="/" class="hover:text-zinc-300 transition-colors">Ana Sayfa</a>
        <span>/</span>
        <a href={`/${art.category || 'haber'}`} class="hover:text-zinc-300 transition-colors uppercase font-semibold">
          {art.category || 'Haber'}
        </a>
        <span>/</span>
        <span class="text-zinc-400 truncate max-w-xs">{art.title}</span>
      </nav>

      <!-- Article Header -->
      <header class="space-y-4 mb-6">
        <div class="flex items-center justify-between flex-wrap gap-2">
          {#if art.category}
            <span class={`inline-block px-3 py-1 rounded ${badge.bg} ${badge.text} text-xs font-bold uppercase tracking-wider shadow`}>
              {art.category}
            </span>
          {/if}
          <div class="flex items-center gap-3 text-xs text-zinc-400">
            <span>⏱ {readingTime} dakika okuma</span>
            <!-- Font sizing controls -->
            <div class="flex items-center bg-zinc-900 border border-zinc-800 rounded-lg p-0.5">
              <button
                type="button"
                onclick={() => changeFontSize(-1)}
                class="px-2 py-0.5 hover:text-white rounded transition-colors text-xs font-bold cursor-pointer"
                title="Yazıyı Küçült"
              >
                A-
              </button>
              <span class="text-zinc-600">|</span>
              <button
                type="button"
                onclick={() => changeFontSize(1)}
                class="px-2 py-0.5 hover:text-white rounded transition-colors text-xs font-bold cursor-pointer"
                title="Yazıyı Büyüt"
              >
                A+
              </button>
            </div>
          </div>
        </div>

        <h1 class="text-2xl sm:text-3xl md:text-5xl font-extrabold text-white leading-tight tracking-tight">
          {art.title}
        </h1>

        {#if art.excerpt}
          <p class="text-base sm:text-lg text-zinc-300 font-medium leading-relaxed border-l-4 border-red-600 pl-4 py-1.5 bg-zinc-900/40 rounded-r-lg">
            {art.excerpt}
          </p>
        {/if}

        <!-- Audio Player Bar (Text to Speech) -->
        {#if speechSupported}
          <div class="flex flex-wrap items-center justify-between gap-3 p-3.5 rounded-xl bg-zinc-900/90 border border-zinc-800 shadow-lg">
            <div class="flex items-center gap-3">
              <button
                type="button"
                onclick={() => toggleSpeech(art)}
                class="w-10 h-10 rounded-full bg-red-600 hover:bg-red-500 text-white flex items-center justify-center font-bold text-lg shadow-lg cursor-pointer transition-all hover:scale-105"
                aria-label="Haberi Dinle"
              >
                {#if isSpeaking && speakingArticleId === art.id && !isPaused}
                  ⏸
                {:else}
                  ▶
                {/if}
              </button>
              <div>
                <div class="text-xs font-bold text-zinc-200">
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
                  class="px-2 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-300 text-xs cursor-pointer"
                >
                  Durdur
                </button>
              {/if}
              <div class="flex items-center gap-1 text-[11px] bg-zinc-950 px-2 py-1 rounded-lg border border-zinc-800 text-zinc-400">
                <span>Hız:</span>
                {#each [1.0, 1.25, 1.5] as rate}
                  <button
                    type="button"
                    onclick={() => changeRate(rate)}
                    class={`px-1.5 py-0.5 rounded ${speechRate === rate ? 'bg-red-600 text-white font-bold' : 'hover:text-zinc-200'}`}
                  >
                    {rate}x
                  </button>
                {/each}
              </div>
            </div>
          </div>
        {/if}

        <div class="flex items-center justify-between py-3 border-y border-zinc-800 text-xs text-zinc-400">
          <div class="flex items-center gap-3 flex-wrap">
            {#if art.author_name}
              <span class="font-bold text-white flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-red-500"></span>
                {art.author_name}
              </span>
            {/if}
            <span class="px-2.5 py-0.5 rounded-full bg-zinc-800/80 border border-zinc-700 text-zinc-300 font-mono text-[11px]">
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
              class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-semibold transition-colors cursor-pointer"
            >
              Paylaş ↗
            </button>
          </div>
        </div>
      </header>

      <!-- Cover Image -->
      <div class="rounded-2xl overflow-hidden bg-zinc-900 border border-zinc-800 mb-8 aspect-video shadow-2xl relative group">
        <img
          src={getMediaUrl(art.cover_media_id, art.category)}
          alt={art.title}
          class="w-full h-full object-cover transition-transform duration-700 group-hover:scale-102"
        />
        <div class="absolute inset-0 bg-gradient-to-t from-black/40 via-transparent to-transparent pointer-events-none"></div>
      </div>

      <!-- In-Article Ad Slot -->
      <AdSlot zone="haber_ici" />

      <!-- Article Body (Quill Delta JSON is automatically transformed to clean HTML) -->
      <article
        class={`prose prose-invert prose-red max-w-none text-zinc-300 leading-relaxed transition-all duration-150 ${
          fontScale === 'xl' ? 'text-2xl leading-loose' : fontScale === 'lg' ? 'text-xl leading-relaxed' : 'sm:text-lg'
        }`}
      >
        {@html sanitizeHtml(art.body)}
      </article>

      <!-- Tags -->
      {#if art.tags}
        {@const tagsList = art.tags.split(',').map((t: string) => t.trim()).filter(Boolean)}
        {#if tagsList.length > 0}
          <div class="mt-8 pt-6 border-t border-zinc-800 flex flex-wrap items-center gap-2">
            <span class="text-xs text-zinc-500 font-medium">Etiketler:</span>
            {#each tagsList as tag}
              <span class="px-2.5 py-1 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-medium text-zinc-300 hover:text-white transition-colors">
                #{tag}
              </span>
            {/each}
          </div>
        {/if}
      {/if}

      <!-- Article Reactions / Reader Feedback -->
      <div class="mt-10 p-5 rounded-2xl bg-zinc-900/60 border border-zinc-800 text-center space-y-3">
        <h3 class="text-sm font-bold text-zinc-300 uppercase tracking-wider">Bu habere tepkiniz nedir?</h3>
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
              class={`flex flex-col items-center gap-1 px-3 py-2 rounded-xl border transition-all cursor-pointer ${
                userReactions[art.id] === r.key
                  ? 'bg-red-600/20 border-red-500 text-white scale-110 shadow-lg'
                  : 'bg-zinc-800/60 border-zinc-700/60 text-zinc-400 hover:border-zinc-500 hover:text-zinc-200'
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
      <div class="inline-flex items-center gap-3 px-5 py-2.5 rounded-full bg-zinc-900 border border-zinc-800 text-zinc-300 text-xs font-semibold shadow-xl">
        <div class="w-4 h-4 border-2 border-red-500 border-t-transparent rounded-full animate-spin"></div>
        <span>Sıradaki haber yükleniyor...</span>
      </div>
    {:else if hasMore}
      <div class="text-xs text-zinc-600 flex items-center justify-center gap-2">
        <span>↓ Aşağı kaydırarak sonraki habere geçin</span>
      </div>
    {:else}
      <div class="text-xs text-zinc-600 py-4">
        ✓ Akıştaki tüm haberleri okudunuz
      </div>
    {/if}
  </div>

  <!-- Related News Grid at Bottom -->
  {#if data.related && data.related.length > 0}
    <section class="mt-8 pt-8 border-t border-zinc-800">
      <h3 class="text-lg font-bold text-white mb-4 flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded bg-red-600"></span>
        Kategorideki Diğer Başlıklar
      </h3>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {#each data.related as r}
          <a
            href={`/${r.category || 'haber'}/${r.slug}`}
            class="group block p-3 rounded-xl bg-zinc-900/60 border border-zinc-800/80 hover:border-zinc-700 hover:-translate-y-1 transition-all duration-300"
          >
            <div class="aspect-video rounded-lg overflow-hidden bg-zinc-800 mb-2.5">
              <img
                src={getMediaUrl(r.cover_media_id, r.category)}
                alt={r.title}
                class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
                loading="lazy"
              />
            </div>
            <h4 class="text-xs font-bold text-zinc-200 line-clamp-2 leading-snug group-hover:text-red-400 transition-colors">
              {r.title}
            </h4>
          </a>
        {/each}
      </div>
    </section>
  {/if}
</div>

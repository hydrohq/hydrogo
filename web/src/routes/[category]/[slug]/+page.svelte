<script lang="ts">
  import type { PageProps } from './$types';
  import { getMediaUrl, sanitizeHtml } from '#lib';
  import AdSlot from '#lib/components/AdSlot.svelte';
  import { onMount } from 'svelte';

  let { data }: PageProps = $props();
  let article = $derived(data.article);

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

  function getCleanArticleText(): string {
    // Strips HTML tags for clean text to be spoken
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = article.body || '';
    const bodyText = tempDiv.textContent || tempDiv.innerText || '';
    return `${article.title}. ${article.excerpt || ''}. ${bodyText}`;
  }

  function toggleSpeech() {
    if (!speechSupported) return;
    const synth = window.speechSynthesis;

    if (isSpeaking) {
      if (isPaused) {
        synth.resume();
        isPaused = false;
      } else {
        synth.pause();
        isPaused = true;
      }
    } else {
      synth.cancel(); // Stop any pending utterances
      const text = getCleanArticleText();
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.lang = article.locale === 'en' ? 'en-US' : 'tr-TR';
      utterance.rate = speechRate;

      utterance.onend = () => {
        isSpeaking = false;
        isPaused = false;
      };

      utterance.onerror = () => {
        isSpeaking = false;
        isPaused = false;
      };

      synth.speak(utterance);
      isSpeaking = true;
      isPaused = false;
    }
  }

  function stopSpeech() {
    if (!speechSupported) return;
    window.speechSynthesis.cancel();
    isSpeaking = false;
    isPaused = false;
  }

  function changeRate(newRate: number) {
    speechRate = newRate;
    if (isSpeaking) {
      // restart with new rate
      stopSpeech();
      toggleSpeech();
    }
  }

  // --- Reactions / Emojis ---
  let userReaction = $state<string | null>(null);
  let reactionCounts = $state<Record<string, number>>({
    like: 42,
    claps: 18,
    wow: 9,
    sad: 3,
    angry: 2,
  });

  function react(type: string) {
    if (userReaction === type) return;
    if (userReaction) {
      reactionCounts[userReaction] = Math.max(0, (reactionCounts[userReaction] || 1) - 1);
    }
    userReaction = type;
    reactionCounts[type] = (reactionCounts[type] || 0) + 1;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(`reaction_${article.id}`, type);
    }
  }

  // --- Estimated Reading Time ---
  let readingTimeMinutes = $derived.by(() => {
    const words = (article.body || '').replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
    return Math.max(1, Math.ceil(words / 180));
  });

  onMount(() => {
    speechSupported = typeof window !== 'undefined' && 'speechSynthesis' in window;

    if (typeof localStorage !== 'undefined') {
      const saved = localStorage.getItem(`reaction_${article.id}`);
      if (saved) userReaction = saved;
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
    return () => {
      window.removeEventListener('scroll', handleScroll);
      if (speechSupported) {
        window.speechSynthesis.cancel();
      }
    };
  });
</script>

<svelte:head>
  <title>{article.seo_title || article.title} — HydroPost</title>
  <meta name="description" content={article.seo_description || article.excerpt || article.title} />
  <!-- OpenGraph -->
  <meta property="og:title" content={article.title} />
  <meta property="og:description" content={article.excerpt || article.title} />
  <meta property="og:type" content="article" />
  {#if article.cover_media_id}
    <meta property="og:image" content={getMediaUrl(article.cover_media_id)} />
  {/if}
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
      class="h-full bg-gradient-to-r from-red-600 to-amber-500 transition-all duration-75"
      style={`width: ${scrollProgress}%`}
    ></div>
  </div>

  {#if showStickyHeader}
    <div class="bg-zinc-950/90 backdrop-blur-md border-b border-zinc-800 px-4 py-2 flex items-center justify-between text-xs text-zinc-300 shadow-xl">
      <div class="flex items-center gap-3 truncate max-w-xl">
        <span class="px-2 py-0.5 rounded bg-red-600 text-white font-bold uppercase text-[10px]">
          {article.category || 'Haber'}
        </span>
        <span class="font-semibold truncate text-white">{article.title}</span>
      </div>

      <div class="flex items-center gap-2 shrink-0">
        <span class="text-zinc-500 hidden sm:inline">{readingTimeMinutes} dk okuma</span>
        {#if speechSupported}
          <button
            type="button"
            onclick={toggleSpeech}
            class="px-2.5 py-1 rounded bg-zinc-800 hover:bg-zinc-700 text-zinc-200 font-medium flex items-center gap-1.5 transition-colors cursor-pointer"
          >
            {#if isSpeaking && !isPaused}
              <span>⏸</span> <span>Duraklat</span>
            {:else if isSpeaking && isPaused}
              <span>▶</span> <span>Devam Et</span>
            {:else}
              <span>🔊</span> <span>Dinle</span>
            {/if}
          </button>
        {/if}
        <button
          type="button"
          onclick={() => {
            if (navigator.share) {
              navigator.share({ title: article.title, url: window.location.href });
            } else {
              navigator.clipboard.writeText(window.location.href);
              alert('Bağlantı kopyalandı!');
            }
          }}
          class="px-2 py-1 rounded bg-red-600 hover:bg-red-500 text-white font-bold cursor-pointer transition-colors"
        >
          Paylaş ↗
        </button>
      </div>
    </div>
  {/if}
</div>

<div class="max-w-4xl mx-auto py-4">
  <!-- Breadcrumbs -->
  <nav class="flex items-center gap-2 text-xs text-zinc-500 mb-4 select-none">
    <a href="/" class="hover:text-zinc-300 transition-colors">Ana Sayfa</a>
    <span>/</span>
    <a href={`/${article.category || 'haber'}`} class="hover:text-zinc-300 transition-colors uppercase">
      {article.category || 'Haber'}
    </a>
    <span>/</span>
    <span class="text-zinc-400 truncate max-w-xs">{article.title}</span>
  </nav>

  <!-- Article Header -->
  <header class="space-y-4 mb-6">
    <div class="flex items-center justify-between">
      {#if article.category}
        <span class="inline-block px-3 py-1 rounded bg-red-600 text-white text-xs font-bold uppercase tracking-wider">
          {article.category}
        </span>
      {/if}
      <div class="flex items-center gap-3 text-xs text-zinc-400">
        <span>⏱ {readingTimeMinutes} dakika okuma</span>
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
      {article.title}
    </h1>

    {#if article.excerpt}
      <p class="text-base sm:text-lg text-zinc-300 font-medium leading-relaxed border-l-4 border-red-600 pl-4 py-1">
        {article.excerpt}
      </p>
    {/if}

    <!-- Audio Player Bar (Text to Speech) -->
    {#if speechSupported}
      <div class="flex flex-wrap items-center justify-between gap-3 p-3.5 rounded-xl bg-zinc-900/80 border border-zinc-800">
        <div class="flex items-center gap-3">
          <button
            type="button"
            onclick={toggleSpeech}
            class="w-10 h-10 rounded-full bg-red-600 hover:bg-red-500 text-white flex items-center justify-center font-bold text-lg shadow-lg cursor-pointer transition-all hover:scale-105"
            aria-label="Haberi Dinle"
          >
            {#if isSpeaking && !isPaused}
              ⏸
            {:else}
              ▶
            {/if}
          </button>
          <div>
            <div class="text-xs font-bold text-zinc-200">
              {isSpeaking ? (isPaused ? 'Sesli Dinleme Duraklatıldı' : 'Sesli Dinleniyor...') : 'Bu Haberi Sesli Dinle'}
            </div>
            <div class="text-[11px] text-zinc-500">Doğal tarayıcı seslendiricisi ile kesintisiz haber akışı</div>
          </div>
        </div>

        <div class="flex items-center gap-2">
          {#if isSpeaking}
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
        {#if article.author_name}
          <span class="font-bold text-white flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full bg-red-500"></span>
            {article.author_name}
          </span>
        {/if}
        <span class="px-2 py-0.5 rounded bg-zinc-800 text-zinc-300 font-mono text-[11px]">
          Mahreç: {article.author_name ? 'Özel Haber' : 'Haber Merkezi'}
        </span>
        <span>Yayınlanma: {new Date(article.created_at).toLocaleDateString('tr-TR', { day: 'numeric', month: 'long', year: 'numeric' })}</span>
      </div>

      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={() => {
            if (navigator.share) {
              navigator.share({ title: article.title, url: window.location.href });
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
  {#if article.cover_media_id}
    <div class="rounded-2xl overflow-hidden bg-zinc-900 border border-zinc-800 mb-8 aspect-video shadow-2xl">
      <img
        src={getMediaUrl(article.cover_media_id)}
        alt={article.title}
        class="w-full h-full object-cover"
      />
    </div>
  {/if}

  <!-- In-Article Ad Slot -->
  <AdSlot zone="haber_ici" />

  <!-- Article Body with Font Scaling -->
  <article
    class={`prose prose-invert prose-red max-w-none text-zinc-300 leading-relaxed transition-all duration-150 ${
      fontScale === 'xl' ? 'text-2xl leading-loose' : fontScale === 'lg' ? 'text-xl leading-relaxed' : 'sm:text-lg'
    }`}
  >
    {@html sanitizeHtml(article.body)}
  </article>

  <!-- Tags -->
  {#if article.tags}
    {@const tagsList = article.tags.split(',').map((t: string) => t.trim()).filter(Boolean)}
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
          onclick={() => react(r.key)}
          class={`flex flex-col items-center gap-1 px-3 py-2 rounded-xl border transition-all cursor-pointer ${
            userReaction === r.key
              ? 'bg-red-600/20 border-red-500 text-white scale-110 shadow-lg'
              : 'bg-zinc-800/60 border-zinc-700/60 text-zinc-400 hover:border-zinc-500 hover:text-zinc-200'
          }`}
        >
          <span class="text-2xl">{r.emoji}</span>
          <span class="text-[11px] font-semibold">{r.label}</span>
          <span class="text-[10px] text-zinc-500 font-bold">{reactionCounts[r.key] || 0}</span>
        </button>
      {/each}
    </div>
  </div>

  <!-- Bottom Ad Slot -->
  <AdSlot zone="footer" class="my-8" />

  <!-- Related News -->
  {#if data.related && data.related.length > 0}
    <section class="mt-12 pt-8 border-t border-zinc-800">
      <h3 class="text-lg font-bold text-white mb-4 flex items-center gap-2">
        <span class="w-2 h-2 rounded bg-red-600"></span>
        İlgili Haberler
      </h3>
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {#each data.related as r}
          <a
            href={`/${r.category || 'haber'}/${r.slug}`}
            class="group block p-3 rounded-xl bg-zinc-900/60 border border-zinc-800/80 hover:border-zinc-700 transition-colors"
          >
            {#if r.cover_media_id}
              <div class="aspect-video rounded-lg overflow-hidden bg-zinc-800 mb-2.5">
                <img
                  src={getMediaUrl(r.cover_media_id)}
                  alt={r.title}
                  class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                  loading="lazy"
                />
              </div>
            {/if}
            <h4 class="text-xs font-bold text-zinc-200 line-clamp-2 leading-snug group-hover:text-red-400 transition-colors">
              {r.title}
            </h4>
          </a>
        {/each}
      </div>
    </section>
  {/if}
</div>

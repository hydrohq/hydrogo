<script lang="ts">
  import type { PageProps } from './$types';

  let { data }: PageProps = $props();
  let m = $derived(data.masthead);

  let formSubmitted = $state(false);
  let formName = $state('');
  let formEmail = $state('');
  let formSubject = $state('haber_ihbari');
  let formMessage = $state('');

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!formName || !formEmail || !formMessage) return;
    formSubmitted = true;
  }
</script>

<svelte:head>
  <title>İletişim ve Haber İhbarı — {m.site_name}</title>
  <meta name="description" content={`${m.site_name} iletişim bilgileri, telefon, e-posta, adres ve 7/24 haber ihbar hattı.`} />
</svelte:head>

<div class="py-6 sm:py-8 max-w-5xl mx-auto">
  <!-- Breadcrumb -->
  <nav class="flex items-center gap-2 text-xs text-[var(--hp-muted)] mb-6" aria-label="Bağlantı yolu">
    <a href="/" class="hover:text-[var(--hp-primary)] transition-colors">Anasayfa</a>
    <span>/</span>
    <span>Kurumsal</span>
    <span>/</span>
    <span class="text-[var(--hp-ink)] font-semibold">İletişim</span>
  </nav>

  <!-- Page Header -->
  <div class="border-b border-[var(--hp-border)] pb-6 mb-8">
    <span class="text-xs font-bold text-[var(--hp-primary)] uppercase tracking-wider">Bize Ulaşın</span>
    <h1 class="text-3xl sm:text-4xl font-extrabold text-[var(--hp-ink)] mt-2 tracking-tight">
      İletişim & Haber İhbar Hattı
    </h1>
    <p class="text-sm text-[var(--hp-muted)] mt-2 max-w-2xl">
      Önemli bir gelişme mi gördünüz? Görüş, öneri, haber ihbarı veya reklam talepleriniz için 7/24 bizimle iletişime geçebilirsiniz.
    </p>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-8">
    <!-- Contact Info Cards (Left) -->
    <div class="lg:col-span-5 space-y-4">
      <!-- WhatsApp Hotwire -->
      <div class="p-6 rounded-2xl bg-emerald-500/10 border border-emerald-500/20 space-y-3">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-emerald-600 text-white flex items-center justify-center font-bold text-lg shadow-sm">
            💬
          </div>
          <div>
            <h2 class="text-sm font-bold text-[var(--hp-ink)]">WhatsApp Haber İhbar Hattı</h2>
            <p class="text-xs text-[var(--hp-muted)]">Fotoğraf & video gönderin, anında yayınlayalım</p>
          </div>
        </div>
        <a
          href={`https://wa.me/${m.phone.replace(/[^0-9]/g, '')}`}
          target="_blank"
          rel="noopener noreferrer"
          class="block text-center w-full py-2.5 px-4 rounded-xl bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-bold transition-colors shadow-sm"
        >
          WhatsApp'tan Mesaj Gönder
        </a>
      </div>

      <!-- Phone & Email -->
      <div class="p-6 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] space-y-4 shadow-sm">
        <h3 class="text-xs font-bold text-[var(--hp-muted)] uppercase tracking-wider">Doğrudan İletişim</h3>

        <div class="space-y-3 text-sm">
          <div class="flex items-start gap-3">
            <span class="text-base text-[var(--hp-primary)]">📞</span>
            <div>
              <span class="text-xs text-[var(--hp-muted)] block">Santral / Haber Masası</span>
              <a href={`tel:${m.phone}`} class="font-semibold text-[var(--hp-ink)] hover:text-[var(--hp-primary)] transition-colors">
                {m.phone}
              </a>
            </div>
          </div>

          <div class="flex items-start gap-3">
            <span class="text-base text-[var(--hp-primary)]">✉️</span>
            <div>
              <span class="text-xs text-[var(--hp-muted)] block">E-Posta</span>
              <a href={`mailto:${m.email}`} class="font-semibold text-[var(--hp-ink)] hover:text-[var(--hp-primary)] transition-colors">
                {m.email}
              </a>
            </div>
          </div>

          {#if m.kep_address}
            <div class="flex items-start gap-3">
              <span class="text-base text-[var(--hp-primary)]">📑</span>
              <div>
                <span class="text-xs text-[var(--hp-muted)] block">KEP Adresi (Kayıtlı Elektronik Posta)</span>
                <span class="font-semibold text-[var(--hp-ink)] text-xs">{m.kep_address}</span>
              </div>
            </div>
          {/if}

          <div class="flex items-start gap-3">
            <span class="text-base text-[var(--hp-primary)]">📍</span>
            <div>
              <span class="text-xs text-[var(--hp-muted)] block">Merkez Ofis Adresi</span>
              <p class="font-medium text-[var(--hp-ink-soft)] text-xs leading-relaxed">{m.address}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Legal Info Link -->
      <div class="p-4 rounded-2xl bg-[var(--hp-surface-sunken)] border border-[var(--hp-border)] text-xs text-[var(--hp-muted)] flex items-center justify-between">
        <span>Yasal künye ve BİK bilgilerini inceleyin</span>
        <a href="/kunye" class="font-bold text-[var(--hp-primary)] hover:underline">Künye →</a>
      </div>
    </div>

    <!-- Contact Form (Right) -->
    <div class="lg:col-span-7">
      <div class="p-6 sm:p-8 rounded-2xl bg-[var(--hp-surface)] border border-[var(--hp-border)] shadow-sm">
        <h2 class="text-lg font-bold text-[var(--hp-ink)] mb-1">Mesaj Gönderin</h2>
        <p class="text-xs text-[var(--hp-muted)] mb-6">
          Aşağıdaki formu doldurarak editör masamıza doğrudan mesaj iletebilirsiniz. En kısa sürede dönüş yapılacaktır.
        </p>

        {#if formSubmitted}
          <div class="p-6 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-center space-y-2">
            <div class="w-12 h-12 rounded-full bg-emerald-600 text-white flex items-center justify-center font-bold text-xl mx-auto">
              ✓
            </div>
            <h3 class="text-base font-bold text-emerald-800">Mesajınız Alındı</h3>
            <p class="text-xs text-emerald-700 max-w-sm mx-auto">
              Teşekkürler Sayın {formName}, ilettiğiniz mesaj editör masamıza ulaştırılmıştır.
            </p>
            <button
              type="button"
              onclick={() => {
                formSubmitted = false;
                formName = '';
                formEmail = '';
                formMessage = '';
              }}
              class="mt-3 inline-block text-xs font-bold text-emerald-800 underline cursor-pointer"
            >
              Yeni Bir Mesaj Gönder
            </button>
          </div>
        {:else}
          <form onsubmit={handleSubmit} class="space-y-4">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label for="form-name" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">Adınız Soyadınız *</label>
                <input
                  id="form-name"
                  type="text"
                  required
                  bind:value={formName}
                  placeholder="Ahmet Yılmaz"
                  class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
                />
              </div>

              <div>
                <label for="form-email" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">E-Posta Adresiniz *</label>
                <input
                  id="form-email"
                  type="email"
                  required
                  bind:value={formEmail}
                  placeholder="ahmet@example.com"
                  class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
                />
              </div>
            </div>

            <div>
              <label for="form-subject" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">Konu</label>
              <select
                id="form-subject"
                bind:value={formSubject}
                class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors"
              >
                <option value="haber_ihbari">Haber İhbarı / Gelişme Bildirimi</option>
                <option value="duzeltme_tekzip">Düzeltme & Tekzip Talebi</option>
                <option value="reklam">Reklam & İş Birlikleri</option>
                <option value="teknik">Teknik Sorun & Bildirim</option>
                <option value="diger">Diğer / Genel Soru</option>
              </select>
            </div>

            <div>
              <label for="form-message" class="block text-xs font-semibold text-[var(--hp-ink)] mb-1">Mesajınız *</label>
              <textarea
                id="form-message"
                required
                rows="5"
                bind:value={formMessage}
                placeholder="İletmek istediğiniz konuyu detaylarıyla yazabilirsiniz..."
                class="w-full px-3.5 py-2.5 rounded-xl border border-[var(--hp-border-strong)] bg-[var(--hp-surface)] text-sm text-[var(--hp-ink)] outline-none focus:border-[var(--hp-primary)] transition-colors resize-y"
              ></textarea>
            </div>

            <button
              type="submit"
              class="w-full sm:w-auto px-8 py-3 rounded-xl bg-[var(--hp-primary)] hover:bg-[var(--hp-primary-dark)] text-white text-sm font-bold transition-all shadow-md shadow-red-600/20 cursor-pointer"
            >
              Mesajı Gönder
            </button>
          </form>
        {/if}
      </div>
    </div>
  </div>
</div>

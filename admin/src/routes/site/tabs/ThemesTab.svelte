<script lang="ts">
  import { onMount } from 'svelte';
  import { t } from '$lib/i18n/index.svelte';
  import { api } from '$lib/api/client';
  import type { ApiResponse } from '$lib/api/types';
  import Button from '$lib/components/ui/Button.svelte';

  export interface ThemeTokens {
    bg: string;
    surface: string;
    fg: string;
    muted: string;
    accent: string;
    font_display: string;
    font_body: string;
    radius: string;
    header_variant: string;
  }

  export interface ThemeManifest {
    id: string;
    name: string;
    description: string;
    supported_presets: string[];
    required_modules?: string[];
    default_tokens: ThemeTokens;
  }

  const FALLBACK_THEMES: ThemeManifest[] = [
    {
      id: 'minimal',
      name: 'Minimal',
      description: 'Clean, fast-loading layout for corporate sites, portfolios, and SaaS landing pages.',
      supported_presets: ['minimal', 'corporate', 'docs', 'saas'],
      default_tokens: {
        bg: '#ffffff',
        surface: '#f8fafc',
        fg: '#0f172a',
        muted: '#64748b',
        accent: '#00add8',
        font_display: 'sans',
        font_body: 'sans',
        radius: 'md',
        header_variant: 'split',
      },
    },
    {
      id: 'boutique',
      name: 'Boutique',
      description: 'Visual-first commerce storefront with editorial hero, lookbook grid, and slide-over cart.',
      supported_presets: ['boutique', 'commerce'],
      required_modules: ['commerce'],
      default_tokens: {
        bg: '#faf8f5',
        surface: '#f2ede4',
        fg: '#1c1917',
        muted: '#78716c',
        accent: '#9a3412',
        font_display: 'serif',
        font_body: 'sans',
        radius: 'sm',
        header_variant: 'centered',
      },
    },
    {
      id: 'editorial',
      name: 'Editorial',
      description: 'Typography-led newsroom and magazine layout with category rails and featured stories.',
      supported_presets: ['news', 'tr-news', 'blog'],
      default_tokens: {
        bg: '#ffffff',
        surface: '#f4f4f5',
        fg: '#18181b',
        muted: '#52525b',
        accent: '#dc2626',
        font_display: 'serif',
        font_body: 'serif',
        radius: 'none',
        header_variant: 'centered',
      },
    },
    {
      id: 'b2b-catalog',
      name: 'B2B Catalog',
      description: 'High-density wholesale catalog with quick-order matrix, SKU search, and dealer pricing.',
      supported_presets: ['erp-lite', 'commerce'],
      required_modules: ['commerce'],
      default_tokens: {
        bg: '#f8fafc',
        surface: '#ffffff',
        fg: '#0f172a',
        muted: '#475569',
        accent: '#2563eb',
        font_display: 'sans',
        font_body: 'sans',
        radius: 'sm',
        header_variant: 'minimal',
      },
    },
  ];

  let {
    settings = $bindable<Record<string, string>>({}),
    saving = false,
    onsubmit,
  }: {
    settings: Record<string, string>;
    saving: boolean;
    onsubmit: (e: SubmitEvent) => void;
  } = $props();

  let themes = $state<ThemeManifest[]>(FALLBACK_THEMES);

  const activeThemeId = $derived(settings['storefront.theme_id'] || 'minimal');
  const activeManifest = $derived(themes.find((item) => item.id === activeThemeId) ?? themes[0]);

  function parseTokens(raw: string | undefined, fallback: ThemeTokens): ThemeTokens {
    if (!raw?.trim()) return { ...fallback };
    try {
      const parsed = JSON.parse(raw) as Partial<ThemeTokens>;
      return { ...fallback, ...parsed };
    } catch {
      return { ...fallback };
    }
  }

  let tokens = $state<ThemeTokens>({ ...FALLBACK_THEMES[0].default_tokens });

  $effect(() => {
    tokens = parseTokens(settings['storefront.theme_tokens'], activeManifest.default_tokens);
  });

  function selectTheme(theme: ThemeManifest) {
    settings['storefront.theme_id'] = theme.id;
    tokens = { ...theme.default_tokens };
    settings['storefront.theme_tokens'] = JSON.stringify(tokens);
  }

  function updateToken<K extends keyof ThemeTokens>(key: K, value: ThemeTokens[K]) {
    tokens = { ...tokens, [key]: value };
    settings['storefront.theme_tokens'] = JSON.stringify(tokens);
  }

  function resetTokens() {
    tokens = { ...activeManifest.default_tokens };
    settings['storefront.theme_tokens'] = '';
  }

  onMount(async () => {
    try {
      const res = await api.get<ApiResponse<{ themes?: ThemeManifest[] }>>('/api/v1/site/themes');
      if (res?.data?.themes && res.data.themes.length > 0) {
        themes = res.data.themes;
      }
    } catch {
      // Keep built-in fallback catalog when offline or in isolated tests.
    }
  });
</script>

<form {onsubmit} class="space-y-6">
  <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-4">
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
      <div>
        <h3 class="text-sm font-semibold text-foreground">{t('site.themes.title')}</h3>
        <p class="text-xs text-muted-foreground mt-0.5">{t('site.themes.subtitle')}</p>
      </div>
      <a
        href={`/?theme_preview=${encodeURIComponent(activeThemeId)}`}
        target="_blank"
        rel="noreferrer"
        class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md border border-input bg-background hover:bg-accent text-xs font-medium text-foreground transition-colors"
      >
        <span>{t('site.themes.preview')}</span>
        <span aria-hidden="true">↗</span>
      </a>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      {#each themes as theme (theme.id)}
        {@const isSelected = activeThemeId === theme.id}
        {@const previewTokens = isSelected ? tokens : theme.default_tokens}
        <div
          class="rounded-xl border p-4 transition-all flex flex-col justify-between gap-3 {isSelected
            ? 'border-primary ring-1 ring-primary/30 bg-primary/5'
            : 'border-border bg-background hover:border-foreground/30'}"
        >
          <div class="space-y-3">
            <!-- Mini Storefront Wireframe Preview -->
            <div
              class="rounded-lg border p-3 space-y-2 overflow-hidden"
              style="background-color: {previewTokens.bg}; color: {previewTokens.fg}; border-color: {previewTokens.surface};"
            >
              <div class="flex items-center justify-between text-[10px] font-semibold">
                {#if previewTokens.header_variant === 'centered'}
                  <span class="opacity-50">Menu</span>
                  <span>Brand</span>
                  <span style="color: {previewTokens.accent};">Cart</span>
                {:else}
                  <span>Brand</span>
                  <div class="flex items-center gap-2 opacity-60">
                    <span>Shop</span>
                    <span>Journal</span>
                    <span style="color: {previewTokens.accent};">Cart</span>
                  </div>
                {/if}
              </div>
              <div
                class="p-2.5 space-y-1.5"
                style="background-color: {previewTokens.surface}; border-radius: {previewTokens.radius === 'none' ? '0px' : previewTokens.radius === 'sm' ? '4px' : previewTokens.radius === 'lg' ? '12px' : '8px'};"
              >
                <div class="h-2 w-24 rounded-xs" style="background-color: {previewTokens.fg}; opacity: 0.85;"></div>
                <div class="h-1.5 w-36 rounded-xs" style="background-color: {previewTokens.muted}; opacity: 0.6;"></div>
                <div class="inline-block h-3 w-12 mt-1" style="background-color: {previewTokens.accent}; border-radius: {previewTokens.radius === 'none' ? '0px' : '4px'};"></div>
              </div>
            </div>

            <div class="flex items-start justify-between gap-2">
              <div>
                <div class="flex items-center gap-2">
                  <span class="text-sm font-semibold text-foreground">{theme.name}</span>
                  {#if isSelected}
                    <span class="rounded-full bg-primary/15 px-2 py-0.5 text-[10px] font-semibold text-primary">
                      {t('site.themes.active')}
                    </span>
                  {/if}
                </div>
                <p class="text-xs text-muted-foreground mt-1 leading-relaxed">{theme.description}</p>
              </div>
            </div>
          </div>

          <div class="flex items-center justify-between gap-2 pt-2 border-t border-border/60">
            <span class="text-[11px] text-muted-foreground truncate">
              {t('site.themes.presetsLabel')} {theme.supported_presets.join(', ')}
            </span>
            <Button
              type="button"
              size="sm"
              variant={isSelected ? 'secondary' : 'outline'}
              onclick={() => selectTheme(theme)}
            >
              {isSelected ? t('site.themes.selected') : t('site.themes.select')}
            </Button>
          </div>
        </div>
      {/each}
    </div>
  </div>

  <!-- Design Token Customizer -->
  <div class="rounded-xl border border-border bg-card p-6 shadow-xs space-y-5">
    <div class="flex items-center justify-between">
      <h3 class="text-sm font-semibold text-foreground">{t('site.themes.tokensTitle')}</h3>
      <button
        type="button"
        onclick={resetTokens}
        class="text-xs text-muted-foreground hover:text-foreground underline underline-offset-2 cursor-pointer"
      >
        {t('site.themes.resetDefaults')}
      </button>
    </div>

    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.accent')}</span>
        <div class="flex items-center gap-2">
          <input
            type="color"
            value={tokens.accent}
            oninput={(e) => updateToken('accent', (e.currentTarget as HTMLInputElement).value)}
            class="h-8 w-10 rounded border border-input bg-background cursor-pointer p-0.5"
          />
          <span class="font-mono text-[11px] text-muted-foreground">{tokens.accent}</span>
        </div>
      </label>

      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.bg')}</span>
        <div class="flex items-center gap-2">
          <input
            type="color"
            value={tokens.bg}
            oninput={(e) => updateToken('bg', (e.currentTarget as HTMLInputElement).value)}
            class="h-8 w-10 rounded border border-input bg-background cursor-pointer p-0.5"
          />
          <span class="font-mono text-[11px] text-muted-foreground">{tokens.bg}</span>
        </div>
      </label>

      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.surface')}</span>
        <div class="flex items-center gap-2">
          <input
            type="color"
            value={tokens.surface}
            oninput={(e) => updateToken('surface', (e.currentTarget as HTMLInputElement).value)}
            class="h-8 w-10 rounded border border-input bg-background cursor-pointer p-0.5"
          />
          <span class="font-mono text-[11px] text-muted-foreground">{tokens.surface}</span>
        </div>
      </label>

      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.fg')}</span>
        <div class="flex items-center gap-2">
          <input
            type="color"
            value={tokens.fg}
            oninput={(e) => updateToken('fg', (e.currentTarget as HTMLInputElement).value)}
            class="h-8 w-10 rounded border border-input bg-background cursor-pointer p-0.5"
          />
          <span class="font-mono text-[11px] text-muted-foreground">{tokens.fg}</span>
        </div>
      </label>
    </div>

    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.radius')}</span>
        <select
          value={tokens.radius}
          onchange={(e) => updateToken('radius', (e.currentTarget as HTMLSelectElement).value)}
          class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground"
        >
          <option value="none">none (0px)</option>
          <option value="sm">sm (4px)</option>
          <option value="md">md (8px)</option>
          <option value="lg">lg (12px)</option>
        </select>
      </label>

      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.fontDisplay')}</span>
        <select
          value={tokens.font_display}
          onchange={(e) => updateToken('font_display', (e.currentTarget as HTMLSelectElement).value)}
          class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground"
        >
          <option value="sans">Sans</option>
          <option value="serif">Serif</option>
          <option value="mono">Mono</option>
          <option value="display">Display</option>
        </select>
      </label>

      <label class="space-y-1.5 text-xs font-medium text-foreground">
        <span>{t('site.themes.headerVariant')}</span>
        <select
          value={tokens.header_variant}
          onchange={(e) => updateToken('header_variant', (e.currentTarget as HTMLSelectElement).value)}
          class="w-full h-9 rounded-md border border-input bg-background px-3 text-xs text-foreground"
        >
          <option value="split">Split</option>
          <option value="centered">Centered</option>
          <option value="minimal">Minimal</option>
        </select>
      </label>
    </div>

    <div class="flex justify-end pt-2 border-t border-border">
      <Button type="submit" disabled={saving}>
        {t('site.themes.save')}
      </Button>
    </div>
  </div>
</form>

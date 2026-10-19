import { api } from '$lib/api/client';
import type { AdminBranding, ApiResponse } from '$lib/api/types';

/** The project's admin name, logo and accent colour. Loaded before sign-in. */
class BrandingState {
  branding = $state<AdminBranding | null>(null);
  defaultLocale = $state<string | null>(null);

  async load(): Promise<void> {
    try {
      const res = await api.get<ApiResponse<AdminBranding & { default_locale?: string }>>('/api/v1/admin/branding');
      if (res?.data) {
        this.branding = { name: res.data.name, logo_url: res.data.logo_url, color: res.data.color };
        this.defaultLocale = res.data.default_locale || null;
      }
    } catch {
      // Branding is cosmetic: the HydroGo defaults apply when it cannot load.
    }
  }

  /** Called after Site settings are saved so the panel updates without a reload. */
  set(branding: AdminBranding) {
    this.branding = branding;
  }
}

export const brandingStore = new BrandingState();

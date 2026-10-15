import { api } from '$lib/api/client';
import type { AdminSchema, Collection, ApiResponse } from '$lib/api/types';

class SchemaState {
  schema = $state<AdminSchema | null>(null);
  isLoading = $state(false);
  error = $state<string | null>(null);

  get resources(): Collection[] {
    return this.schema?.resources || [];
  }

  get collections(): Collection[] {
    return this.resources.filter(r => r.source === 'content_type' || !r.source);
  }

  get modules(): Record<string, boolean> {
    return this.schema?.modules || {};
  }

  get locales(): string[] {
    return this.schema?.locales?.all || ['en'];
  }

  get defaultLocale(): string {
    return this.schema?.locales?.default || 'en';
  }

  getCollection(id: string): Collection | undefined {
    return this.resources.find(r => r.id === id);
  }

  async loadSchema(force = false): Promise<void> {
    if (this.schema && !this.error && !force) return;
    this.isLoading = true;
    this.error = null;
    try {
      const res = await api.get<ApiResponse<AdminSchema>>('/api/v1/admin/schema');
      if (res && res.data) {
        this.schema = res.data;
      }
    } catch (err: any) {
      this.error = err.message || 'Failed to load admin schema';
      console.error('Schema load error:', err);
    } finally {
      this.isLoading = false;
    }
  }
}

export const schema = new SchemaState();

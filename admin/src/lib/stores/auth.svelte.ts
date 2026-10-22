import { api, setCsrfToken } from '$lib/api/client';
import { router } from '$lib/router.svelte';
import type { User, MeResponse, ApiResponse } from '$lib/api/types';

class AuthState {
  user = $state<User | null>(null);
  permissions = $state<string[]>([]);
  /** The admin language saved server-side (from /me); null when none. */
  serverLocale = $state<string | null>(null);
  isLoading = $state(true);
  error = $state<string | null>(null);

  get isAuthenticated(): boolean {
    return this.user !== null && this.user.active;
  }

  hasPermission(perm: string): boolean {
    if (!this.user) return false;
    if (this.user.role === 'admin') return true;
    return this.permissions.includes(perm);
  }

  async checkAuth(): Promise<boolean> {
    this.isLoading = true;
    this.error = null;
    try {
      const res = await api.get<ApiResponse<MeResponse>>('/api/v1/admin/me');
      if (res && res.data) {
        this.user = res.data.user;
        this.permissions = res.data.permissions || [];
        this.serverLocale = res.data.locale ?? null;
        if (res.data.csrf_token) {
          setCsrfToken(res.data.csrf_token);
        }
        this.isLoading = false;
        return true;
      }
    } catch (err: any) {
      this.user = null;
      this.permissions = [];
      this.serverLocale = null;
      this.error = err.message;
    } finally {
      this.isLoading = false;
    }
    return false;
  }

  async login(payload: { email: string; password: string; code?: string }): Promise<boolean> {
    this.isLoading = true;
    this.error = null;
    try {
      const res = await api.post<any>('/auth/login', payload);
      if (res) {
        if (res.mfa_required) {
          return false;
        }
        await this.checkAuth();
        return true;
      }
    } catch (err: any) {
      this.error = err.message;
      throw err;
    } finally {
      this.isLoading = false;
    }
    return false;
  }

  async logout(): Promise<void> {
    try {
      await api.post('/auth/logout');
    } catch (e) {
      console.warn('Logout error', e);
    } finally {
      this.user = null;
      this.permissions = [];
      this.serverLocale = null;
      router.navigate('/login');
    }
  }
}

export const auth = new AuthState();

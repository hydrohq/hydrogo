import { api } from '$lib/api/client';
import type { ApiResponse } from '$lib/api/types';

export interface HookService {
  id: string;
  host: string;
  timeout?: string;
  schedules: { name: string; cron: string }[];
  used_by: string[];
  calls: number;
  failures: number;
  last_at?: string;
  last_status?: number;
  last_error?: string;
  last_event?: string;
}

/** The configured hook services; empty when none are configured (the endpoint then does not exist). */
export async function loadHookServices(): Promise<HookService[]> {
  try {
    const res = await api.get<ApiResponse<{ services: HookService[] }>>('/api/v1/admin/system/services');
    return res?.data?.services ?? [];
  } catch {
    return [];
  }
}

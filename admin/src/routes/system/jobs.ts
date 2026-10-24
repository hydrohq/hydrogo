import { api } from '$lib/api/client';
import type { ApiResponse } from '$lib/api/types';

export interface JobSummary {
  id: string;
  type: string;
  payload: string;
  available_at: string;
  created_at: string;
  attempts: number;
  dead_lettered: boolean;
  last_error?: string;
  lease_owner?: string;
  lease_until?: string;
}

export async function loadJobs(): Promise<JobSummary[]> {
  const res = await api.get<ApiResponse<{ jobs: JobSummary[] }>>('/api/v1/admin/jobs');
  return res?.data?.jobs ?? [];
}

export async function loadDeadLetterJobs(): Promise<JobSummary[]> {
  const res = await api.get<ApiResponse<{ jobs: JobSummary[] }>>('/api/v1/admin/jobs/dead-letters');
  return res?.data?.jobs ?? [];
}

export async function requeueJob(id: string): Promise<void> {
  await api.post(`/api/v1/admin/jobs/dead-letters/${encodeURIComponent(id)}/requeue`);
}

export async function deleteDeadLetterJob(id: string): Promise<void> {
  await api.delete(`/api/v1/admin/jobs/dead-letters/${encodeURIComponent(id)}`);
}

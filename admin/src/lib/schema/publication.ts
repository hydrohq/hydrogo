import { api } from '$lib/api/client';

// A `drafts: true` type (no custom state machine) keeps its status server-side: PUT/POST saves ignore any
// `status` in the payload, so publishing / unpublishing must go through POST <endpoint>/<id>/workflow.
type Draftish = { drafts?: boolean; workflow?: { enabled?: boolean; state_field?: string; states?: string[] } } | null | undefined;

// The server describes a drafts type as a workflow over a `status` field (states draft + published); the plain
// `drafts` flag is not always sent, so recognise both.
export const usesDrafts = (collection?: Draftish): boolean => {
  if (collection?.workflow?.enabled) return false;
  const states = collection?.workflow?.states ?? [];
  return Boolean(collection?.drafts) || (states.includes('draft') && states.includes('published') && Boolean(collection?.workflow?.state_field));
};

/** The raw status field is managed by the publish buttons, so forms and lists hide it for drafts types. */
export const hiddenStatusField = (collection?: Draftish): string => (usesDrafts(collection) ? collection?.workflow?.state_field || 'status' : '');

/** Moves a record to `target` (`published` or `draft`) unless it is already there. Returns the new status. */
export async function setPublication(endpoint: string, id: string, target: 'published' | 'draft', known?: { status?: string; version?: number }): Promise<string> {
  let status = known?.status;
  let version = known?.version;
  if (status === undefined || !version) {
    const current = (await api.get<{ data: Record<string, any> }>(`${endpoint}/${id}`)).data;
    status = current.status;
    version = Number(current.version) || 0;
  }
  if ((status || 'draft') === target) return target;
  await api.post(`${endpoint}/${id}/workflow`, { target, version });
  return target;
}

import { t, type MessageKey } from './index.svelte';

// Known enum values map to catalog keys; anything else (custom roles, new
// statuses from the API) is shown as received.
const STATUS_KEYS: Record<string, MessageKey> = {
  published: 'common.published',
  draft: 'common.draft',
  scheduled: 'common.scheduled',
  retracted: 'common.retracted',
};

const ROLE_KEYS: Record<string, MessageKey> = {
  admin: 'enum.role.admin',
  editor: 'enum.role.editor',
  author: 'enum.role.author',
  viewer: 'enum.role.viewer',
};

const KIND_KEYS: Record<string, MessageKey> = {
  category: 'enum.kind.category',
  tag: 'enum.kind.tag',
};

function label(map: Record<string, MessageKey>, value: string): string {
  const key = Object.hasOwn(map, value) ? map[value] : undefined;
  return key ? t(key) : value;
}

export const statusLabel = (value: string) => label(STATUS_KEYS, value);
export const roleLabel = (value: string) => label(ROLE_KEYS, value);
export const kindLabel = (value: string) => label(KIND_KEYS, value);

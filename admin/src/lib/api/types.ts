// HydroGo Admin API TypeScript Definitions

export interface ApiResponse<T> {
  data: T;
  /** The API envelope is {data, metadata}; `meta` is the older spelling. */
  metadata?: {
    total?: number;
    limit?: number;
    offset?: number;
    has_more?: boolean;
    [key: string]: unknown;
  };
  meta?: {
    total?: number;
    page?: number;
    per_page?: number;
    [key: string]: unknown;
  };
}

export interface ApiError {
  code: string;
  message: string;
  request_id: string;
  fields?: Record<string, string>;
}

export interface User {
  id: string;
  email: string;
  role: 'admin' | 'editor' | 'author' | 'member' | string;
  active: boolean;
  mfa_enabled?: boolean;
  locale?: string;
}

export interface MeResponse {
  user: User;
  permissions: string[];
  csrf_token: string;
  locales?: string[];
  /** Admin language saved on the server; null until the user chose one. */
  locale?: string | null;
}

export type FieldKind =
  | 'text'
  | 'textarea'
  | 'richtext'
  | 'number'
  | 'boolean'
  | 'date'
  | 'datetime'
  | 'select'
  | 'media'
  | 'relation';

export interface SelectOption {
  label: string;
  value: string | number;
}

export interface FieldRelation {
  collection: string;
  /** "one" or "many", as the server sends it. */
  cardinality?: 'one' | 'many';
  many?: boolean;
}

export interface Field {
  id?: string;
  name?: string;
  label: string;
  kind: FieldKind;
  widget?: string;
  required?: boolean;
  unique?: boolean;
  list?: boolean;
  visible_in_list?: boolean;
  searchable?: boolean;
  filterable?: boolean;
  localized?: boolean;
  read_only?: boolean;
  options?: (SelectOption | string)[];
  relation?: FieldRelation;
  default?: unknown;
}

export interface CollectionWorkflow {
  enabled: boolean;
  drafts: boolean;
  states: string[];
  initial: string;
}

export interface CollectionPermissions {
  view?: string;
  create?: string;
  edit?: string;
  delete?: string;
}

export interface CollectionStateMachine {
  field: string;
  initial: string;
  transitions: { from: string; to: string; permission?: string }[];
}

export interface Collection {
  id: string;
  label: string;
  singular?: string;
  plural?: string;
  description?: string;
  group?: string;
  icon?: string;
  source?: 'content_type' | 'module' | string;
  api_path?: string;
  title_field?: string;
  versioned?: boolean;
  revisions?: boolean;
  drafts?: boolean;
  localization?: boolean;
  workflow?: CollectionWorkflow;
  state_machine?: CollectionStateMachine;
  fields: Field[];
  permissions?: CollectionPermissions;
}

export interface AdminSchemaContentType {
  id: string;
  label: string;
  description: string;
  public: boolean;
  drafts: boolean;
  revisions: boolean;
  localization: boolean;
  title_field?: string;
}

export interface AdminSchemaLocales {
  enabled: boolean;
  default: string;
  all: string[];
  /** Project-wide admin panel language set in Site settings (empty = none). */
  admin_default?: string;
}

export interface AdminBranding {
  name: string;
  logo_url: string;
  color: string;
}

export interface AdminSchema {
  version: number;
  resources: Collection[];
  content_types: AdminSchemaContentType[];
  modules: Record<string, boolean>;
  locales: AdminSchemaLocales;
  permissions: string[];
  branding?: AdminBranding;
}

export interface ContentItem {
  id: string;
  status?: 'draft' | 'published' | 'archived' | string;
  created_at?: string;
  updated_at?: string;
  locale?: string;
  [key: string]: unknown;
}

export interface MediaItem {
  id: string;
  name: string;
  path: string;
  size: number;
  content_type: string;
  folder?: string;
  created_at?: string;
}

export interface TenantGovernance {
  tenant_id: string;
  display_name: string;
  subdomain: string;
  custom_domain: string;
  blueprint: string;
  plan: 'starter' | 'pro' | 'vip' | 'custom' | string;
  lifecycle_state: 'provisioning' | 'active' | 'readonly' | 'suspended' | 'archived' | string;
  effective_state: 'provisioning' | 'active' | 'readonly' | 'suspended' | 'archived' | string;
  trial_ends_at?: string;
  subscription_ends_at?: string;
  max_storage_bytes: number;
  max_records: number;
  rate_limit_rps: number;
  max_concurrent_reqs: number;
  max_upload_bytes: number;
  ai_enabled: boolean;
  ai_provider: string;
  ai_model: string;
  has_custom_api_key: boolean;
  ai_monthly_token_limit: number;
  ai_tokens_used: number;
  owner_phone: string;
  notes: string;
  created_at: string;
  updated_at: string;
}

export interface TenantUsage {
  tenant_id: string;
  storage_bytes: number;
  max_storage_bytes: number;
  storage_usage_percent: number;
  total_media_assets: number;
  total_records: number;
  max_records_per_collection: number;
}

export interface WorkspaceSummary {
  governance: TenantGovernance;
  usage: TenantUsage;
  admin_email?: string;
  record_usage_percent: number;
  ai_token_usage_percent: number;
}

export interface OrchestratorTotals {
  total_workspaces: number;
  active_workspaces: number;
  readonly_workspaces: number;
  suspended_workspaces: number;
  archived_workspaces: number;
  total_storage_bytes: number;
  total_records: number;
  total_ai_tokens_used: number;
}

export interface OrchestratorListResponse {
  workspaces: WorkspaceSummary[];
  totals: OrchestratorTotals;
}

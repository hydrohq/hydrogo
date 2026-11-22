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
  value_field?: string;
}

export interface CollectionAction {
  label: string;
  href?: string;
  copy_text?: string;
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
  private?: boolean;
  localized?: boolean;
  read_only?: boolean;
  options?: (SelectOption | string)[];
  relation?: FieldRelation;
  default?: unknown;
}

export interface CollectionWorkflow {
  enabled?: boolean;
  drafts?: boolean;
  state_field?: string;
  states: string[];
  initial?: string;
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
  type?: 'collection' | 'global' | string;
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
  actions?: CollectionAction[];
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
  platform?: boolean;
  site_url?: string;
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

export interface CopilotOperation {
  op: 'create' | 'update' | 'delete' | string;
  collection_id: string;
  record_id?: string;
  data?: Record<string, unknown>;
  before_data?: Record<string, unknown>;
  existed?: boolean;
}

export interface CopilotActionLog {
  id: string;
  tenant_id: string;
  prompt: string;
  reply: string;
  provider: string;
  model: string;
  tokens_used: number;
  operations: CopilotOperation[];
  undone: boolean;
  prompt_id?: string;
  prompt_version?: string;
  prompt_hash?: string;
  created_at: string;
  undone_at?: string;
}

export interface SitePlan {
  brief: string;
  theme: {
    preset: string;
    font_preset: string;
    bg_color?: string;
    text_color?: string;
    accent_color?: string;
    border_radius?: string;
  };
  layout: Array<{ type: string; props?: Record<string, unknown> }>;
  store_settings: Record<string, unknown>;
  products: Array<{
    title: string;
    slug: string;
    price: number;
    description?: string;
    badge?: string;
    image_url?: string;
  }>;
  prompt_id?: string;
  prompt_version?: string;
  prompt_hash?: string;
}

export interface SuperadminMetrics {
  total_stores: number;
  active_stores: number;
  demo_stores: number;
  suspended_stores: number;
  total_orders: number;
  ram_alloc_bytes: number;
  ram_sys_bytes: number;
  uptime_seconds: number;
  base_domain: string;
  console_host: string;
  server_public_ip: string;
  cloudflare_wildcard_status: string;
  cloudflare_status_message: string;
  credits_remaining?: number;
  credits_total?: number;
  credits_used?: number;
  onboard_invite_code?: string;
  onboard_enabled?: boolean;
}

export interface PlatformCredits {
  account_id: string;
  total_credits: number;
  used_credits: number;
  remaining_credits: number;
  plan: string;
}

export interface SuperadminWorkspace {
  tenant_id: string;
  slug: string;
  name: string;
  live_url: string;
  subdomain: string;
  custom_domain: string;
  dns_verified: boolean;
  status: 'DEMO' | 'ACTIVE' | 'SUSPENDED' | string;
  plan: string;
  owner_phone: string;
  order_count: number;
  created_at: string;
  expires_at?: string;
}

// SaaS subscription and billing types
export interface BillingPlan {
  id: string;
  name: string;
  code: string;
  description: string;
  price_cents: number;
  currency: string;
  billing_interval: 'month' | 'year';
  interval_count: number;
  trial_days: number;
  features: string[];
  active: boolean;
}

export interface BillingOrder {
  id: string;
  user_id: string;
  plan_id: string;
  order_number: string;
  status: 'pending' | 'completed' | 'failed' | 'cancelled';
  amount_cents: number;
  currency: string;
  billing_interval: string;
  idempotency_key?: string;
  notes?: string;
  created_at: string;
}

export interface BillingPayment {
  id: string;
  order_id: string;
  user_id: string;
  payment_provider: string;
  provider_payment_id?: string;
  amount_cents: number;
  currency: string;
  status: 'pending' | 'success' | 'failed';
  error_message?: string;
  paid_at?: string;
  created_at: string;
}

export interface BillingSubscription {
  id: string;
  user_id: string;
  plan_id: string;
  status: 'pending' | 'active' | 'past_due' | 'cancelled' | 'expired';
  current_period_start: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
  cancelled_at?: string;
  plan?: BillingPlan;
  days_remaining: number;
  is_active: boolean;
}

export interface BillingOrderWithPayment {
  order: BillingOrder;
  plan?: BillingPlan;
  payments: BillingPayment[];
}

export interface ShopierFormOutput {
  action_url: string;
  method: string;
  fields: Record<string, string>;
}

export interface BillingCheckoutResponse {
  order: BillingOrder;
  payment: BillingPayment;
  shopier_form: ShopierFormOutput;
}

export interface RefundRequired {
  id: string;
  order_id: string;
  provider: string;
  amount_cents: number;
  currency: string;
  reason: string;
  created_at: string;
  resolved_at?: string | null;
  resolved_note?: string;
}

export interface DiscountCode {
  id: string;
  code: string;
  kind: 'percent' | 'fixed';
  value: number;
  min_subtotal_cents: number;
  max_redemptions: number;
  redeemed_count: number;
  active: boolean;
  starts_at: string;
  ends_at: string;
}

export interface RefundsRequiredCount {
  count: number;
}


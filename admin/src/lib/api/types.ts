// HydroGo Admin API TypeScript Definitions

export interface ApiResponse<T> {
  data: T;
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
  many?: boolean;
}

export interface Field {
  name: string;
  label: string;
  kind: FieldKind;
  required?: boolean;
  unique?: boolean;
  list?: boolean;
  searchable?: boolean;
  filterable?: boolean;
  localized?: boolean;
  options?: SelectOption[];
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
  view: string;
  create: string;
  edit: string;
  delete: string;
}

export interface Collection {
  id: string;
  label: string;
  singular: string;
  plural: string;
  description: string;
  group: string;
  icon?: string;
  api_path: string;
  workflow?: CollectionWorkflow;
  fields: Field[];
  permissions: CollectionPermissions;
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
}

export interface AdminSchema {
  version: number;
  resources: Collection[];
  content_types: AdminSchemaContentType[];
  modules: Record<string, boolean>;
  locales: AdminSchemaLocales;
  permissions: string[];
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

export interface CatalogSuggestion {
  item_id: string;
  name?: string;
  score: number;
  reason: string;
}

export interface CatalogItem {
  id: string;
  name: string;
  kind: string;
  status: string;
  internal_sku: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateCatalogItemInput {
  id: string;
  name: string;
  kind: string;
  internal_sku: string;
}

export interface UpdateCatalogItemInput {
  name?: string;
  kind?: string;
  status?: string;
  internal_sku?: string;
}

export interface CatalogReviewLine {
  id: string;
  invoice_header_id: string;
  line_number: number;
  item_code: string;
  description: string;
  item_id: string | null;
  link_status: string;
  link_method: string | null;
  link_locked: boolean;
  suggestions: CatalogSuggestion[];
}

export interface LineDecisionPayload {
  item_id?: string;
  action: 'link' | 'never_match' | 'create_provisional';
  remember: boolean;
  lock: boolean;
}

export const CATALOG_KINDS = [
  { value: 'goods', label: 'Bien' },
  { value: 'service', label: 'Servicio' },
  { value: 'asset', label: 'Activo' },
  { value: 'unknown', label: 'Desconocido' },
] as const;

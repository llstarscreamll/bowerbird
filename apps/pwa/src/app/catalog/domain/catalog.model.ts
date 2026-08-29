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

export const CATALOG_KINDS = [
  { value: 'goods', label: 'Bien' },
  { value: 'service', label: 'Servicio' },
  { value: 'asset', label: 'Activo' },
  { value: 'unknown', label: 'Desconocido' },
] as const;

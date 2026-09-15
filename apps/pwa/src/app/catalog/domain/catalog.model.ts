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
  creation_source: string;
  internal_code: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateCatalogItemInput {
  id: string;
  name: string;
  kind: string;
  internal_code: string;
}

export interface UpdateCatalogItemInput {
  name?: string;
  kind?: string;
  status?: string;
  internal_code?: string;
}

export const CATALOG_KINDS = [
  { value: 'goods', label: 'Bien' },
  { value: 'service', label: 'Servicio' },
  { value: 'asset', label: 'Activo' },
  { value: 'unknown', label: 'Desconocido' },
] as const;

export function creationSourceLabel(source: string): string {
  switch (source) {
    case 'manual':
      return 'Manual';
    case 'invoice':
      return 'Desde factura';
    case 'import':
      return 'Importación';
    default:
      return source;
  }
}

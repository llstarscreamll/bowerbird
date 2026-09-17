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
  aliases?: CatalogAlias[];
}

export interface CatalogAlias {
  id: string;
  scheme: 'supplier_sku' | 'gtin' | string;
  value: string;
  party_id?: string | null;
  source: 'invoice' | 'manual' | string;
}

export interface CreateCatalogAliasInput {
  id: string;
  scheme: string;
  value: string;
  party_id?: string;
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

export interface MergeItemsInput {
  survivor_id: string;
  source_ids: string[];
  name?: string;
  kind?: string;
  internal_code?: string;
}

export interface DuplicateClusterMember {
  id: string;
  name: string;
  kind: string;
  status: string;
  creation_source: string;
  internal_code: string | null;
  created_at: string;
  updated_at: string;
  line_count: number;
  aliases?: CatalogAlias[];
}

export interface DuplicateCluster {
  id: string;
  reason: string;
  item_ids: string[];
  items: DuplicateClusterMember[];
}

export function clusterReasonLabel(reason: string): string {
  switch (reason) {
    case 'normalized_description':
      return 'Misma descripción';
    case 'hard_conflict':
      return 'Conflicto duro en factura (GTIN vs SKU)';
    case 'cross_party_sku':
      return 'Mismo SKU de proveedor en distintos contactos';
    default:
      return reason;
  }
}

export function pickDefaultSurvivor(items: Array<Pick<CatalogItem, 'id' | 'status' | 'internal_code' | 'creation_source' | 'created_at'> & { line_count?: number }>): string {
  const ranked = [...items].sort((a, b) => survivorScore(b) - survivorScore(a));
  return ranked[0]?.id ?? '';
}

function survivorScore(item: Pick<CatalogItem, 'status' | 'internal_code' | 'creation_source' | 'created_at'> & { line_count?: number }): number {
  let score = 0;
  if (item.status === 'confirmed') score += 10_000;
  if (item.internal_code) score += 1_000;
  if (item.creation_source === 'manual' || item.creation_source === 'import') score += 100;
  score += (item.line_count ?? 0) * 10;
  const created = Date.parse(item.created_at);
  if (!Number.isNaN(created)) score += Math.max(0, 2_000_000_000_000 - created) / 1_000_000;
  return score;
}

export function kindLabel(kind: string): string {
  return CATALOG_KINDS.find((row) => row.value === kind)?.label || kind;
}

export function statusLabel(status: string): string {
  switch (status) {
    case 'provisional':
      return 'Provisional';
    case 'confirmed':
      return 'Confirmado';
    case 'merged':
      return 'Fusionado';
    default:
      return status;
  }
}

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

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

export const CATALOG_MERGE_MIN_ITEMS = 2;
export const CATALOG_MERGE_MAX_ITEMS = 20;

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

type SurvivorCandidate = Pick<CatalogItem, 'id' | 'status' | 'internal_code' | 'creation_source' | 'created_at'>;

export function previewMergeInternalCode(survivor: Pick<CatalogItem, 'internal_code'>, items: Array<Pick<CatalogItem, 'id' | 'internal_code'>>, chosenCodeItemId: string): string | null {
  if (survivor.internal_code) return survivor.internal_code;

  const distinctCodes = [...new Set(items.map((item) => item.internal_code).filter((code): code is string => Boolean(code)))];

  if (distinctCodes.length > 1) {
    return items.find((item) => item.id === chosenCodeItemId)?.internal_code ?? null;
  }

  return distinctCodes[0] ?? null;
}

export function isMergeInternalCodeInherited(survivor: Pick<CatalogItem, 'internal_code'>, items: Array<Pick<CatalogItem, 'internal_code'>>, resolvedCode: string | null): boolean {
  if (!resolvedCode || survivor.internal_code) return false;
  const distinctCodes = [...new Set(items.map((item) => item.internal_code).filter((code): code is string => Boolean(code)))];
  return distinctCodes.length === 1 && distinctCodes[0] === resolvedCode;
}

export function pickDefaultSurvivor(items: SurvivorCandidate[]): string {
  const ranked = [...items].sort((a, b) => survivorScore(b) - survivorScore(a));
  return ranked[0]?.id ?? '';
}

export function survivorSuggestionTooltip(winner: SurvivorCandidate, allItems: SurvivorCandidate[]): string {
  const others = allItems.filter((item) => item.id !== winner.id);
  if (others.length === 0) return 'Único ítem del grupo.';

  const reasons: string[] = [];
  if (winner.status === 'confirmed' && others.some((item) => item.status !== 'confirmed')) {
    reasons.push('es el único confirmado');
  } else if (winner.status === 'confirmed') {
    reasons.push('está confirmado');
  }

  if (winner.internal_code && others.some((item) => !item.internal_code)) {
    reasons.push('tiene código interno');
  }

  const winnerCurated = winner.creation_source === 'manual' || winner.creation_source === 'import';
  if (winnerCurated && others.every((item) => item.creation_source === 'invoice')) {
    reasons.push(winner.creation_source === 'manual' ? 'fue creado manualmente' : 'proviene de carga masiva');
  }

  if (reasons.length === 0) return 'Desempate por ser el registro más antiguo del grupo.';
  return `Sugerido porque ${joinReasons(reasons)}.`;
}

function joinReasons(reasons: string[]): string {
  if (reasons.length === 1) return reasons[0];
  if (reasons.length === 2) return `${reasons[0]} y ${reasons[1]}`;
  return `${reasons.slice(0, -1).join(', ')} y ${reasons[reasons.length - 1]}`;
}

function survivorScore(item: SurvivorCandidate): number {
  let score = 0;
  if (item.status === 'confirmed') score += 10_000;
  if (item.internal_code) score += 1_000;
  if (item.creation_source === 'manual' || item.creation_source === 'import') score += 100;
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
      return 'Carga masiva';
    default:
      return source;
  }
}

export interface InvoiceSummary {
  id: string;
  source_name: string;
  source_id: string;
  cufe: string;
  invoice_number: string;
  issuer_name: string;
  issuer_tax_id: string;
  issuer_party_id?: string | null;
  receiver_name: string;
  receiver_tax_id: string;
  currency_code: string;
  issue_date: string | null;
  due_date: string | null;
  payment_code: string;
  subtotal: number;
  tax_total: number;
  allowance_total: number;
  grand_total: number;
  extraction_source: string;
  linking_status?: string;
  created_at: string;
}

export interface InvoiceListResponse {
  items: InvoiceSummary[];
  has_more: boolean;
  limit: number;
  cursor?: string;
}

export interface InvoiceLine {
  id: string;
  line_number: number;
  buyer_code: string;
  seller_sku: string;
  gtin: string;
  description: string;
  quantity: number;
  unit_price: number;
  line_tax_total: number;
  line_total: number;
  item_id?: string | null;
  item_name?: string | null;
  item_internal_code?: string | null;
  link_status?: string;
  link_method?: string | null;
  link_locked?: boolean;
  suggestions?: { item_id: string; name?: string; score: number; reason: string }[];
}

export interface InvoiceDetails extends InvoiceSummary {
  lines: InvoiceLine[];
}

export interface InvoiceReviewLine {
  id: string;
  invoice_header_id: string;
  line_number: number;
  buyer_code: string;
  seller_sku: string;
  gtin: string;
  description: string;
  item_id: string | null;
  link_status: string;
  link_method: string | null;
  link_locked: boolean;
  suggestions: { item_id: string; name?: string; score: number; reason: string }[];
}

export interface LineDecisionPayload {
  item_id?: string;
  action: 'link' | 'never_match' | 'create_provisional' | 'unlock';
  remember: boolean;
  lock: boolean;
}

/** Catalog item fields invoices needs to pick a link target. */
export interface CatalogSearchHit {
  id: string;
  name: string;
  status: string;
}

const unusableSeller = new Set(['1', '01', '001', 'n/a', 'na', 'serv', 'servicio', 'item']);

export function sellerSkuUsable(code: string): boolean {
  const v = (code || '').trim();
  if (v.length < 3) return false;
  return !unusableSeller.has(v.toLowerCase());
}

export function classifyGtin(raw: string): string {
  const digits = (raw || '').replace(/\s/g, '');
  if (!/^\d+$/.test(digits)) return '';
  return [8, 12, 13, 14].includes(digits.length) ? digits : '';
}

export function rememberPreview(sellerSku: string, gtin: string): { seller?: string; gtin?: string } {
  const out: { seller?: string; gtin?: string } = {};
  if (sellerSkuUsable(sellerSku)) out.seller = sellerSku.trim();
  const classified = classifyGtin(gtin);
  if (classified) out.gtin = classified;
  return out;
}

// JSON:API Types
export interface JsonApiDocument<T> {
  type: string;
  id: string;
  attributes: T;
}

export interface JsonApiCollectionMeta {
  total_count?: number;
  limit: number;
  offset?: number;
  cursor?: string;
  has_more: boolean;
}

export interface JsonApiCollectionResponse<T> {
  data: JsonApiDocument<T>[];
  meta: JsonApiCollectionMeta;
}

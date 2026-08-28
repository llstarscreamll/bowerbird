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
  item_code: string;
  description: string;
  quantity: number;
  unit_price: number;
  line_tax_total: number;
  line_total: number;
  item_id?: string | null;
  link_status?: string;
  link_method?: string | null;
  link_locked?: boolean;
  suggestions?: { item_id: string; name?: string; score: number; reason: string }[];
}

export interface InvoiceDetails extends InvoiceSummary {
  lines: InvoiceLine[];
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

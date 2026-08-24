export interface CatalogItem {
  id: string;
  name: string;
  kind: string;
  status: string;
  stockable: boolean | null;
  created_at: string;
  updated_at: string;
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
  suggestions: { item_id: string; score: number; reason: string }[];
}

export interface LineDecisionPayload {
  item_id?: string;
  action: 'link' | 'never_match';
  remember: boolean;
  lock: boolean;
}

export const LEGAL_ENTITY_SCHEMES = [
  { value: '31', label: 'NIT' },
  { value: '13', label: 'Cédula' },
] as const;

export type LegalEntitySchemeId = (typeof LEGAL_ENTITY_SCHEMES)[number]['value'];

export interface LegalEntity {
  id: string;
  tax_id: string;
  scheme_id: string;
  legal_name: string;
  created_at: string;
  updated_at: string;
}

export interface CreateLegalEntityInput {
  tax_id: string;
  scheme_id: string;
  legal_name: string;
}

export interface UpdateLegalEntityInput {
  tax_id?: string;
  scheme_id?: string;
  legal_name?: string;
}

export function schemeLabel(schemeId: string): string {
  return LEGAL_ENTITY_SCHEMES.find((item) => item.value === schemeId)?.label ?? schemeId;
}

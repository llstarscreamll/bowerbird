export interface PartyEmail {
  id: string;
  value: string;
  kind: string;
  source: string;
}

export interface PartyPhone {
  id: string;
  value: string;
  source: string;
}

export interface PartyAddress {
  id: string;
  line: string;
  city: string;
  department: string;
  postal_zone: string;
  country_code: string;
  kind: string;
  source: string;
}

export interface Party {
  id: string;
  tax_id: string;
  scheme_id: string;
  taxpayer_kind: string;
  tax_level_codes: string[];
  name: string;
  roles: string[];
  status: string;
  creation_source: string;
  emails: PartyEmail[];
  phones: PartyPhone[];
  addresses: PartyAddress[];
  created_at: string;
  updated_at: string;
}

export interface CreatePartyInput {
  name: string;
  tax_id: string;
  scheme_id?: string;
  roles: string[];
}

export interface UpdatePartyInput {
  name?: string;
  roles?: string[];
  scheme_id?: string;
  taxpayer_kind?: string;
}

export const PARTY_ROLES = [
  { value: 'supplier', label: 'Proveedor' },
  { value: 'customer', label: 'Cliente' },
] as const;

export const DOCUMENT_SCHEMES = [
  { value: '31', label: 'NIT' },
  { value: '13', label: 'Cédula' },
] as const;

export function roleLabel(role: string): string {
  return PARTY_ROLES.find((r) => r.value === role)?.label ?? role;
}

export function schemeLabel(scheme: string): string {
  return DOCUMENT_SCHEMES.find((s) => s.value === scheme)?.label ?? (scheme || '—');
}

export function taxpayerKindLabel(kind: string): string {
  switch (kind) {
    case '1':
      return 'Persona jurídica';
    case '2':
      return 'Persona natural';
    default:
      return '—';
  }
}

export function creationSourceLabel(source: string): string {
  switch (source) {
    case 'manual':
      return 'Manual';
    case 'invoice':
      return 'Desde factura';
    default:
      return source;
  }
}

export function emailKindLabel(kind: string): string {
  return kind === 'tax_mailbox' ? 'Buzón tributario' : 'General';
}

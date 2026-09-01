export interface Party {
  id: string;
  tax_id: string;
  name: string;
  roles: string[];
  status: string;
  creation_source: string;
  created_at: string;
  updated_at: string;
}

export interface CreatePartyInput {
  name: string;
  tax_id: string;
  roles: string[];
}

export interface UpdatePartyInput {
  name?: string;
  roles?: string[];
}

export const PARTY_ROLES = [
  { value: 'supplier', label: 'Proveedor' },
  { value: 'customer', label: 'Cliente' },
] as const;

export function roleLabel(role: string): string {
  return PARTY_ROLES.find((r) => r.value === role)?.label ?? role;
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

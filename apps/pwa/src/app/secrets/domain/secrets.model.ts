export const SecretPurposes = {
  InvoicingDocumentPassword: 'invoicing.document_password',
  IntegrationsApiKey: 'integrations.api_key',
  IntegrationsIdpCredential: 'integrations.idp_credential',
  GenericCredential: 'generic.credential',
} as const;

export type SecretPurpose = (typeof SecretPurposes)[keyof typeof SecretPurposes];

export interface SecretPurposeDefinition {
  purpose: SecretPurpose;
  label: string;
  shortLabel: string;
  description: string;
  valueLabel: string;
  labelPlaceholder: string;
  valuePlaceholder: string;
}

/** Catalog of secret types the organization can store for system use. */
export const SECRET_PURPOSE_CATALOG: SecretPurposeDefinition[] = [
  {
    purpose: SecretPurposes.IntegrationsApiKey,
    label: 'Clave de API',
    shortLabel: 'API',
    description: 'Tokens o API keys para conectar Bowerbird con el ERP u otros sistemas del cliente.',
    valueLabel: 'API key o token',
    labelPlaceholder: 'ERP producción / Contabilidad',
    valuePlaceholder: 'sk_live_…',
  },
  {
    purpose: SecretPurposes.IntegrationsIdpCredential,
    label: 'Credencial de identidad (IdP)',
    shortLabel: 'IdP',
    description: 'Client secrets OAuth/SAML u otras claves del proveedor de identidad del cliente.',
    valueLabel: 'Client secret o clave',
    labelPlaceholder: 'Azure AD / Okta / Google Workspace',
    valuePlaceholder: '••••••••',
  },
  {
    purpose: SecretPurposes.InvoicingDocumentPassword,
    label: 'Contraseña de documentos',
    shortLabel: 'Documentos',
    description: 'NIT, cédula u otra clave para abrir PDF o ZIP protegidos (por ejemplo, facturas DIAN).',
    valueLabel: 'Contraseña',
    labelPlaceholder: 'NIT Acme / Cédula representante',
    valuePlaceholder: '••••••••',
  },
  {
    purpose: SecretPurposes.GenericCredential,
    label: 'Otra credencial',
    shortLabel: 'Otro',
    description: 'Cualquier otro secreto que el sistema deba usar para acceder a datos o servicios externos.',
    valueLabel: 'Valor del secreto',
    labelPlaceholder: 'Nombre del sistema o uso',
    valuePlaceholder: '••••••••',
  },
];

export function purposeDefinition(purpose: string): SecretPurposeDefinition | undefined {
  return SECRET_PURPOSE_CATALOG.find((item) => item.purpose === purpose);
}

export function purposeLabel(purpose: string): string {
  return purposeDefinition(purpose)?.label ?? purpose;
}

export function purposeShortLabel(purpose: string): string {
  return purposeDefinition(purpose)?.shortLabel ?? purpose;
}

export interface Secret {
  id: string;
  purpose: string;
  kind: string;
  label: string;
  description?: string;
  version: number;
  has_value: boolean;
  last_used_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface SecretResource {
  type: string;
  id: string;
  attributes: Omit<Secret, 'id'>;
}

export interface SecretCollectionResponse {
  data: SecretResource[];
}

export interface SecretDocumentResponse {
  data: SecretResource;
}

export function mapSecret(resource: SecretResource): Secret {
  return {
    id: resource.id,
    purpose: resource.attributes.purpose,
    kind: resource.attributes.kind,
    label: resource.attributes.label,
    description: resource.attributes.description,
    version: resource.attributes.version,
    has_value: resource.attributes.has_value,
    last_used_at: resource.attributes.last_used_at,
    created_at: resource.attributes.created_at,
    updated_at: resource.attributes.updated_at,
  };
}

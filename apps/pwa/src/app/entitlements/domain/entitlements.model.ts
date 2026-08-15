export const FeatureKeys = {
  InvoicingWorkspace: 'invoicing.workspace',
  InvoicingCaptureFromEmail: 'invoicing.capture_from_email',
  MailInbox: 'mail.inbox',
  MailOrganize: 'mail.organize',
  MailSend: 'mail.send',
} as const;

export type FeatureKey = (typeof FeatureKeys)[keyof typeof FeatureKeys];

export interface TenantEntitlements {
  features: string[];
}

export interface PlatformTenant {
  id: string;
  name: string;
  slug: string;
  status: string;
}

export interface EntitlementGrant {
  feature_key: string;
  status: string;
  source: string;
  starts_at: string;
  ends_at?: string | null;
}

export interface TenantEntitlementsDetail {
  tenant_id: string;
  features: string[];
  grants: EntitlementGrant[];
}

export interface SetAccessPayload {
  product?: string;
  feature?: string;
  enabled: boolean;
  ends_at?: string | null;
}

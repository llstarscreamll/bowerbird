export interface AuthTokens {
  access_token: string;
  expires_in: number;
}

export interface User {
  id: string;
  email: string;
}

export interface TenantMembership {
  tenant_id: string;
  name: string;
  role: string;
}

export interface CurrentUser {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  picture_url: string;
  platform_operator: boolean;
}

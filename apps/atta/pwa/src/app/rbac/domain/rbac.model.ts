export const PermissionCodes = {
  SecretsRead: 'secrets:read',
  SecretsWrite: 'secrets:write',
  SecretsDelete: 'secrets:delete',
} as const;

export type PermissionCode = (typeof PermissionCodes)[keyof typeof PermissionCodes];

export interface MyPermissions {
  permissions: string[];
}

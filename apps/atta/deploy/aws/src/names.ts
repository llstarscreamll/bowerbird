export function resourcePrefix(envName: string): string {
  return `${envName}-atta`;
}

export function defaultTags(envName: string): Record<string, string> {
  return {
    Project: 'atta',
    Environment: envName,
    ManagedBy: 'pulumi',
    DeploymentTarget: 'aws-lambda',
  };
}

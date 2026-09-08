export function resourcePrefix(envName: string): string {
  return `${envName}-bowerbird`;
}

export function defaultTags(envName: string): Record<string, string> {
  return {
    Project: 'bowerbird',
    Environment: envName,
    ManagedBy: 'pulumi',
    DeploymentTarget: 'aws-lambda',
  };
}

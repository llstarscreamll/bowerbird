export type LocalUserCredentials = {
  email: string;
  password: string;
};

const randomPart = (): string => Math.random().toString(36).slice(2, 10);

export const buildLocalUserCredentials = (overrides: Partial<LocalUserCredentials> = {}): LocalUserCredentials => {
  const runId = process.env.E2E_RUN_ID ?? String(Date.now());

  return {
    email: overrides.email ?? `e2e.${runId}.${randomPart()}@example.com`,
    password: overrides.password ?? 'P4ssword!e2e',
  };
};

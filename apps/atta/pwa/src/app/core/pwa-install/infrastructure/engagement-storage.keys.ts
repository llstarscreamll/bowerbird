const VISITS_KEY = 'bb:pwa:visits';
const PREFS_KEY = 'bb:pwa:install-prefs';
const SESSION_KEY = 'bb:pwa:session';

export const ENGAGEMENT_STORAGE_KEYS = {
  visits: VISITS_KEY,
  prefs: PREFS_KEY,
  session: SESSION_KEY,
} as const;

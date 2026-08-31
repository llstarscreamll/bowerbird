export type InstallEngagementDomainEvent =
  | { type: 'SessionVisitRecorded'; visitNumber: number }
  | { type: 'AutoPromptBecameEligible'; visitNumber: number }
  | { type: 'AutoPromptDeclined'; reason: string };

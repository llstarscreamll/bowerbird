export const ENGAGEMENT_WINDOW_DAYS = 7;

export class EngagementWindow {
  static readonly days = ENGAGEMENT_WINDOW_DAYS;

  static isWithinWindow(firstVisit: Date, secondVisit: Date): boolean {
    const diffMs = secondVisit.getTime() - firstVisit.getTime();
    return diffMs <= ENGAGEMENT_WINDOW_DAYS * 24 * 60 * 60 * 1000;
  }
}

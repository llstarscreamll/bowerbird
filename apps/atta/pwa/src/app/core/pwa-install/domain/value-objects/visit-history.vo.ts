import { EngagementWindow } from './engagement-window.vo';

export class VisitHistory {
  private constructor(private readonly visits: readonly Date[]) {}

  static empty(): VisitHistory {
    return new VisitHistory([]);
  }

  static reconstitute(timestamps: string[]): VisitHistory {
    return new VisitHistory(timestamps.map((value) => new Date(value)));
  }

  recordVisit(now: Date): VisitHistory {
    return new VisitHistory([...this.visits, now]);
  }

  visitCount(): number {
    return this.visits.length;
  }

  hasMetEngagementThreshold(now: Date): boolean {
    if (this.visits.length < 2) {
      return false;
    }

    const first = this.visits[0];
    const second = this.visits[1];
    return EngagementWindow.isWithinWindow(first, second) && second.getTime() <= now.getTime();
  }

  toISOStrings(): string[] {
    return this.visits.map((visit) => visit.toISOString());
  }
}

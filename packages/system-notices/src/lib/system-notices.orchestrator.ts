import type { SystemNotice, SystemNoticeHandle, SystemNoticeScope, SystemNoticesObserver } from './system-notice.port';

export class SystemNoticesOrchestrator {
  private activeNoticeId: string | null = null;

  constructor(
    private readonly notices: SystemNotice[],
    private readonly observer?: SystemNoticesObserver,
  ) {}

  tryPresent(scope: SystemNoticeScope): void {
    if (this.activeNoticeId) {
      return;
    }

    const candidates = this.notices.filter((notice) => notice.scope === scope).sort((a, b) => b.priority - a.priority);

    for (const notice of candidates) {
      if (!notice.canShow()) {
        continue;
      }

      try {
        this.activeNoticeId = notice.id;
        const handle = this.createHandle(notice.id);
        notice.show(handle);
        this.observer?.onShown?.(notice.id, scope);
        return;
      } catch {
        this.activeNoticeId = null;
      }
    }
  }

  clearActive(noticeId: string, reason: string): void {
    if (this.activeNoticeId !== noticeId) {
      return;
    }

    this.activeNoticeId = null;
    this.observer?.onDismissed?.(noticeId, reason);
  }

  get activeId(): string | null {
    return this.activeNoticeId;
  }

  private createHandle(noticeId: string): SystemNoticeHandle {
    return {
      clearActive: (reason) => this.clearActive(noticeId, reason),
      tryPresent: (nextScope) => this.tryPresent(nextScope),
    };
  }
}

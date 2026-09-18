export type SystemNoticeScope = 'global' | 'tenant';

export interface SystemNoticeHandle {
  clearActive(reason: string): void;
  tryPresent(scope: SystemNoticeScope): void;
}

export interface SystemNotice {
  readonly id: string;
  readonly priority: number;
  readonly scope: SystemNoticeScope;
  canShow(): boolean;
  show(handle: SystemNoticeHandle): void;
  dismiss(reason: string, handle: SystemNoticeHandle): void;
}

export interface SystemNoticesObserver {
  onShown?(noticeId: string, scope: SystemNoticeScope): void;
  onDismissed?(noticeId: string, reason: string): void;
}

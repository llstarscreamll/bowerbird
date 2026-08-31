export type DismissReasonCode = 'timeout' | 'not_now' | 'continue_browser';

export class DismissReason {
  private constructor(readonly code: DismissReasonCode) {}

  static fromUserAction(code: string): DismissReason {
    if (code === 'timeout' || code === 'not_now' || code === 'continue_browser') {
      return new DismissReason(code);
    }
    return new DismissReason('not_now');
  }

  isExplicit(): boolean {
    return this.code === 'not_now' || this.code === 'continue_browser';
  }
}

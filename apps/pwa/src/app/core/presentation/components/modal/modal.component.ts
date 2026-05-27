import { CommonModule } from '@angular/common';
import { Component, ElementRef, EventEmitter, Input, OnChanges, AfterViewInit, Output, SimpleChanges, ViewChild } from '@angular/core';

@Component({
  selector: 'app-modal',
  standalone: true,
  imports: [CommonModule],
  template: `
    <dialog
      #dialog
      closedby="any"
      aria-labelledby="modal-title"
      class="m-auto transform overflow-hidden rounded-2xl bg-white text-left shadow-xl transition-all w-[calc(100vw-2rem)] sm:w-full sm:max-w-lg p-0 backdrop:bg-slate-900/40 backdrop:backdrop-blur-sm focus:outline-none"
      (click)="onDialogClick($event)"
      (cancel)="$event.preventDefault(); emitClose()"
      (close)="emitClose()"
    >
      <div class="bg-white px-4 pb-4 pt-5 sm:p-6 sm:pb-4 w-full">
        <div class="sm:flex sm:items-start">
          <div class="mt-3 text-center sm:mt-0 sm:text-left w-full">
            <h3 class="text-lg font-semibold leading-6 text-slate-900" id="modal-title">
              {{ title }}
            </h3>
            <div class="mt-2 text-sm text-slate-500">
              <ng-content></ng-content>
            </div>
          </div>
        </div>
      </div>

      <div class="bg-slate-50 px-4 py-3 sm:flex sm:flex-row-reverse sm:px-6 gap-3">
        <ng-content select="[modal-footer]">
          <button *ngIf="showDefaultFooter" type="button" class="btn-secondary mt-3 inline-flex w-full sm:mt-0 sm:w-auto" (click)="emitClose()">Cerrar</button>
        </ng-content>
      </div>
    </dialog>
  `,
})
export class ModalComponent implements OnChanges, AfterViewInit {
  @Input() isOpen = false;
  @Input() title = '';
  @Input() showDefaultFooter = true;
  @Output() close = new EventEmitter<void>();

  @ViewChild('dialog') dialogRef!: ElementRef<HTMLDialogElement>;

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['isOpen']) {
      this.syncState();
    }
  }

  ngAfterViewInit(): void {
    this.syncState();
  }

  private syncState() {
    const dialog = this.dialogRef?.nativeElement;
    if (dialog) {
      if (this.isOpen) {
        if (!dialog.open) {
          dialog.showModal();
        }
      } else {
        if (dialog.open) {
          dialog.close();
        }
      }
    }
  }

  // Fallback for light-dismiss
  onDialogClick(event: MouseEvent) {
    const dialog = this.dialogRef.nativeElement;

    // Ignore clicks where the target is a child element inside the dialog.
    if (event.target !== dialog) return;

    // Check if the click coordinates fall within the dialog's content box.
    const rect = dialog.getBoundingClientRect();
    const isDialogContent = rect.top <= event.clientY && event.clientY <= rect.top + rect.height && rect.left <= event.clientX && event.clientX <= rect.left + rect.width;

    if (!isDialogContent) {
      this.emitClose();
    }
  }

  emitClose() {
    this.close.emit();
  }
}

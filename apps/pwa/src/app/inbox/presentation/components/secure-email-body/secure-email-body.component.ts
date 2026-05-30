import { CommonModule } from '@angular/common';
import { Component, computed, input, signal } from '@angular/core';
import { secureEmailHtml } from '../../../application/email-html-security';

@Component({
  selector: 'app-secure-email-body',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="space-y-3">
      @if (blockedExternalImages() > 0 && !showExternalImages()) {
        <div class="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900">
          Para proteger tu privacidad, se han bloqueado {{ blockedExternalImages() }} imágenes externas.
          <button type="button" class="ml-2 font-semibold underline underline-offset-2" (click)="enableExternalImages()">Mostrar imágenes</button>
        </div>
      }

      <iframe
        class="w-full h-[75dvh] min-h-[420px] rounded-lg border border-slate-200 bg-white"
        [attr.srcdoc]="iframeSrcDoc()"
        sandbox="allow-popups allow-popups-to-escape-sandbox"
        referrerpolicy="no-referrer"
        title="Contenido del correo"
      ></iframe>
    </div>
  `,
})
export class SecureEmailBodyComponent {
  readonly html = input<string>('');

  readonly showExternalImages = signal(false);
  private readonly secured = computed(() => secureEmailHtml(this.html(), this.showExternalImages()));

  readonly blockedExternalImages = computed(() => this.secured().blockedExternalImages);
  readonly iframeSrcDoc = computed(() => {
    const content = this.secured().sanitizedHtml;
    return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><style>body{font-family:ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;color:#1f2937;background:#fff;margin:0;padding:16px;line-height:1.5}img{max-width:100%;height:auto}a{color:#2563eb;text-decoration:underline}</style></head><body>${content}</body></html>`;
  });

  enableExternalImages(): void {
    this.showExternalImages.set(true);
  }
}

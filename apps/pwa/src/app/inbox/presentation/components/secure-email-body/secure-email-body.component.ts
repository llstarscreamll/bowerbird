import { Component, computed, inject, input, signal } from '@angular/core';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { ThemeService } from '../../../../core/services/theme.service';
import { secureEmailHtml } from '../../../application/email-html-security';

const EMAIL_IFRAME_STYLES = {
  light: {
    background: '#ffffff',
    foreground: '#0f172a',
    link: '#4f46e5',
  },
  dark: {
    background: '#0f172a',
    foreground: '#f8fafc',
    link: '#818cf8',
  },
} as const;

@Component({
  selector: 'app-secure-email-body',
  standalone: true,
  imports: [NgIcon, HlmAlertImports, HlmButtonImports],
  template: `
    <div class="space-y-3">
      @if (blockedExternalImages() > 0 && !showExternalImages()) {
        <hlm-alert>
          <ng-icon hlm name="lucideTriangleAlert" />
          <p hlmAlertDescription>
            Para proteger tu privacidad, se han bloqueado {{ blockedExternalImages() }} imágenes externas.
            <button type="button" hlmBtn variant="link" class="h-auto p-0" (click)="enableExternalImages()">Mostrar imágenes</button>
          </p>
        </hlm-alert>
      }

      <iframe
        class="h-[75dvh] min-h-[420px] w-full rounded-lg border bg-card"
        [attr.srcdoc]="iframeSrcDoc()"
        sandbox="allow-popups allow-popups-to-escape-sandbox"
        referrerpolicy="no-referrer"
        title="Contenido del correo"
      ></iframe>
    </div>
  `,
})
export class SecureEmailBodyComponent {
  private readonly themeService = inject(ThemeService);

  readonly html = input<string>('');

  readonly showExternalImages = signal(false);
  private readonly secured = computed(() => secureEmailHtml(this.html(), this.showExternalImages()));

  readonly blockedExternalImages = computed(() => this.secured().blockedExternalImages);
  readonly iframeSrcDoc = computed(() => {
    const palette = this.themeService.isDark() ? EMAIL_IFRAME_STYLES.dark : EMAIL_IFRAME_STYLES.light;
    const content = this.secured().sanitizedHtml;

    return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><style>body{font-family:ui-sans-serif,system-ui,-apple-system,"Segoe UI",sans-serif;color:${palette.foreground};background:${palette.background};margin:0;padding:16px;line-height:1.5}img{max-width:100%;height:auto}a{color:${palette.link};text-decoration:underline}</style></head><body>${content}</body></html>`;
  });

  enableExternalImages(): void {
    this.showExternalImages.set(true);
  }
}

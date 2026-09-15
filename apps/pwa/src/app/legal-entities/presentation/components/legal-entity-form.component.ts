import { Component, effect, input, output } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CreateLegalEntityInput, LEGAL_ENTITY_SCHEMES, LegalEntity } from '../../domain/legal-entity.model';

export type LegalEntityFormValue = CreateLegalEntityInput;

@Component({
  selector: 'app-legal-entity-form',
  standalone: true,
  imports: [ReactiveFormsModule, HlmButtonImports, HlmSpinnerImports],
  host: { class: 'block' },
  template: `
    <form class="flex flex-col gap-4" [formGroup]="form" (ngSubmit)="onSubmit()">
      <fieldset class="flex flex-col gap-2">
        <legend class="text-sm font-medium">Tipo de identificación</legend>
        <div class="flex flex-col gap-2">
          @for (scheme of schemes; track scheme.value) {
            <label class="inline-flex items-center gap-2 text-sm" [attr.for]="'legal-entity-scheme-' + scheme.value">
              <input type="radio" [id]="'legal-entity-scheme-' + scheme.value" name="scheme_id" formControlName="scheme_id" [value]="scheme.value" required />
              {{ scheme.label }}
            </label>
          }
        </div>
      </fieldset>

      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="legal-entity-tax-id">NIT o cédula</label>
        <input
          id="legal-entity-tax-id"
          name="tax_id"
          formControlName="tax_id"
          required
          autocomplete="off"
          inputmode="numeric"
          enterkeyhint="next"
          aria-describedby="legal-entity-tax-id-help"
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-base focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:text-sm"
        />
        <p id="legal-entity-tax-id-help" class="text-xs text-muted-foreground">Solo dígitos. No uses el dígito de verificación.</p>
      </div>

      <div class="flex flex-col gap-1.5">
        <label class="text-sm font-medium" for="legal-entity-legal-name">Razón social</label>
        <input
          id="legal-entity-legal-name"
          name="legal_name"
          formControlName="legal_name"
          required
          autocomplete="organization"
          enterkeyhint="done"
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-base focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:text-sm"
        />
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button hlmBtn type="submit" [disabled]="submitting()">
          @if (submitting()) {
            <hlm-spinner class="size-4" />
          }
          {{ submitLabel() }}
        </button>
      </div>
    </form>
  `,
})
export class LegalEntityFormComponent {
  readonly initial = input<LegalEntity | null>(null);
  readonly submitting = input(false);
  readonly submitLabel = input('Guardar');
  readonly submitted = output<LegalEntityFormValue>();

  readonly schemes = LEGAL_ENTITY_SCHEMES;

  private readonly fb = new FormBuilder();
  readonly form = this.fb.nonNullable.group({
    tax_id: ['', Validators.required],
    scheme_id: ['31', Validators.required],
    legal_name: ['', Validators.required],
  });

  constructor() {
    effect(() => {
      const entity = this.initial();
      if (entity) {
        this.form.patchValue({
          tax_id: entity.tax_id,
          scheme_id: entity.scheme_id,
          legal_name: entity.legal_name,
        });
      }
    });
  }

  onSubmit(): void {
    if (this.form.invalid || this.submitting()) return;
    const raw = this.form.getRawValue();
    this.submitted.emit({
      tax_id: raw.tax_id.trim(),
      scheme_id: raw.scheme_id,
      legal_name: raw.legal_name.trim(),
    });
  }
}

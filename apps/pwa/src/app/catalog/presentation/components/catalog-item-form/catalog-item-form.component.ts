import { Component, effect, input, output } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CATALOG_KINDS, CatalogItem } from '../../../domain/catalog.model';

export type CatalogItemFormMode = 'create' | 'edit' | 'confirm';

export interface CatalogItemFormValue {
  name: string;
  kind: string;
  internal_sku: string;
}

@Component({
  selector: 'app-catalog-item-form',
  standalone: true,
  imports: [ReactiveFormsModule, HlmButtonImports, HlmSpinnerImports],
  host: { class: 'block' },
  template: `
    <form class="space-y-4" [formGroup]="form" (ngSubmit)="onSubmit()">
      <div class="space-y-1.5">
        <label class="text-sm font-medium" for="catalog-item-name">Nombre</label>
        <input
          id="catalog-item-name"
          name="name"
          formControlName="name"
          required
          autocomplete="off"
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium" for="catalog-item-kind">Tipo</label>
        <select
          id="catalog-item-kind"
          name="kind"
          formControlName="kind"
          required
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        >
          @for (k of kinds; track k.value) {
            <option [value]="k.value">{{ k.label }}</option>
          }
        </select>
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium" for="catalog-item-sku">SKU interno</label>
        <input
          id="catalog-item-sku"
          name="internal_sku"
          formControlName="internal_sku"
          autocomplete="off"
          [required]="skuRequired"
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-60"
        />
        @if (skuReadonly) {
          <p class="text-xs text-muted-foreground">El SKU interno no se puede cambiar una vez asignado.</p>
        } @else if (mode() === 'confirm') {
          <p class="text-xs text-muted-foreground">Obligatorio para confirmar un ítem provisional.</p>
        }
      </div>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button hlmBtn type="button" variant="outline" [disabled]="submitting()" (click)="cancelled.emit()">Cancelar</button>
        <button hlmBtn type="submit" [disabled]="form.invalid || submitting()">
          @if (submitting()) {
            <hlm-spinner class="size-4" />
          }
          {{ submitLabel }}
        </button>
      </div>
    </form>
  `,
})
export class CatalogItemFormComponent {
  readonly mode = input<CatalogItemFormMode>('create');
  readonly initial = input<CatalogItem | null>(null);
  readonly submitting = input(false);
  readonly submitted = output<CatalogItemFormValue>();
  readonly cancelled = output<void>();

  readonly kinds = CATALOG_KINDS;

  private readonly fb = new FormBuilder();
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    kind: ['goods', Validators.required],
    internal_sku: [''],
  });

  constructor() {
    effect(() => {
      const item = this.initial();
      if (item) {
        this.form.patchValue({
          name: item.name,
          kind: item.kind || 'unknown',
          internal_sku: item.internal_sku ?? '',
        });
      }
      this.applySkuRules();
    });
  }

  get skuReadonly(): boolean {
    return this.mode() !== 'create' && !!this.initial()?.internal_sku;
  }

  get skuRequired(): boolean {
    return this.mode() === 'create' || this.mode() === 'confirm' || !this.initial()?.internal_sku;
  }

  get submitLabel(): string {
    if (this.mode() === 'create') return 'Crear ítem';
    if (this.mode() === 'confirm') return 'Confirmar ítem';
    return 'Guardar cambios';
  }

  private applySkuRules(): void {
    const ctrl = this.form.controls.internal_sku;
    if (this.skuReadonly) {
      ctrl.disable({ emitEvent: false });
      ctrl.clearValidators();
    } else {
      ctrl.enable({ emitEvent: false });
      ctrl.setValidators(this.skuRequired ? [Validators.required] : []);
    }
    ctrl.updateValueAndValidity({ emitEvent: false });
  }

  onSubmit(): void {
    if (this.form.invalid || this.submitting()) return;
    const raw = this.form.getRawValue();
    this.submitted.emit({
      name: raw.name.trim(),
      kind: raw.kind,
      internal_sku: raw.internal_sku.trim(),
    });
  }
}

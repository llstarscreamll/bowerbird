import { Component, effect, input, output } from '@angular/core';
import { FormBuilder, ReactiveFormsModule, Validators } from '@angular/forms';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { PARTY_ROLES, Party } from '../../../domain/party.model';

export type PartyFormMode = 'create' | 'edit';

export interface PartyFormValue {
  name: string;
  tax_id: string;
  roles: string[];
}

@Component({
  selector: 'app-party-form',
  standalone: true,
  imports: [ReactiveFormsModule, HlmButtonImports, HlmSpinnerImports],
  host: { class: 'block' },
  template: `
    <form class="space-y-4" [formGroup]="form" (ngSubmit)="onSubmit()">
      <div class="space-y-1.5">
        <label class="text-sm font-medium" for="party-name">Nombre</label>
        <input
          id="party-name"
          name="name"
          formControlName="name"
          required
          autocomplete="organization"
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        />
      </div>

      <div class="space-y-1.5">
        <label class="text-sm font-medium" for="party-tax-id">NIT</label>
        <input
          id="party-tax-id"
          name="tax_id"
          formControlName="tax_id"
          required
          autocomplete="off"
          class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-60"
        />
        @if (mode() === 'edit') {
          <p class="text-xs text-muted-foreground">El NIT no se puede cambiar una vez creado.</p>
        }
      </div>

      <fieldset class="space-y-2">
        <legend class="text-sm font-medium">Roles</legend>
        <div class="flex flex-col gap-2">
          @for (role of roleOptions; track role.value) {
            <label class="inline-flex items-center gap-2 text-sm">
              <input type="checkbox" [checked]="hasRole(role.value)" (change)="toggleRole(role.value, $event)" />
              {{ role.label }}
            </label>
          }
        </div>
        @if (rolesTouched && selectedRoles.length === 0) {
          <p class="text-xs text-destructive">Selecciona al menos un rol.</p>
        }
      </fieldset>

      <div class="flex items-center justify-end gap-2 pt-2">
        <button hlmBtn type="button" variant="outline" [disabled]="submitting()" (click)="cancelled.emit()">Cancelar</button>
        <button hlmBtn type="submit" [disabled]="!canSubmit">
          @if (submitting()) {
            <hlm-spinner class="size-4" />
          }
          {{ submitLabel }}
        </button>
      </div>
    </form>
  `,
})
export class PartyFormComponent {
  readonly mode = input<PartyFormMode>('create');
  readonly initial = input<Party | null>(null);
  readonly submitting = input(false);
  readonly submitted = output<PartyFormValue>();
  readonly cancelled = output<void>();

  readonly roleOptions = PARTY_ROLES;
  selectedRoles: string[] = [];
  rolesTouched = false;

  private readonly fb = new FormBuilder();
  readonly form = this.fb.nonNullable.group({
    name: ['', Validators.required],
    tax_id: ['', Validators.required],
  });

  constructor() {
    effect(() => {
      this.mode();
      const party = this.initial();
      if (party) {
        this.form.patchValue({ name: party.name, tax_id: party.tax_id });
        this.selectedRoles = [...party.roles];
      } else {
        this.selectedRoles = [];
      }
      this.applyTaxIdRules();
    });
  }

  get submitLabel(): string {
    return this.mode() === 'create' ? 'Crear contacto' : 'Guardar cambios';
  }

  get canSubmit(): boolean {
    return this.form.valid && this.selectedRoles.length > 0 && !this.submitting();
  }

  hasRole(role: string): boolean {
    return this.selectedRoles.includes(role);
  }

  toggleRole(role: string, event: Event): void {
    this.rolesTouched = true;
    const checked = (event.target as HTMLInputElement).checked;
    if (checked) {
      if (!this.selectedRoles.includes(role)) {
        this.selectedRoles = [...this.selectedRoles, role];
      }
    } else {
      this.selectedRoles = this.selectedRoles.filter((r) => r !== role);
    }
  }

  onSubmit(): void {
    this.rolesTouched = true;
    if (!this.canSubmit) return;
    const raw = this.form.getRawValue();
    this.submitted.emit({
      name: raw.name.trim(),
      tax_id: raw.tax_id.trim(),
      roles: [...this.selectedRoles],
    });
  }

  private applyTaxIdRules(): void {
    const ctrl = this.form.controls.tax_id;
    if (this.mode() === 'edit') {
      ctrl.disable({ emitEvent: false });
    } else {
      ctrl.enable({ emitEvent: false });
    }
  }
}

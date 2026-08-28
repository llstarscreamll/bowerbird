import { Component, OnInit, computed, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { CatalogStore } from '../../../application/catalog.store';
import { UpdateCatalogItemInput } from '../../../domain/catalog.model';
import { CatalogItemFormComponent, CatalogItemFormMode, CatalogItemFormValue } from '../../components/catalog-item-form/catalog-item-form.component';

@Component({
  selector: 'app-catalog-item-edit',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmCardImports, HlmAlertImports, HlmSpinnerImports, CatalogItemFormComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al detalle
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">{{ formMode() === 'confirm' ? 'Confirmar ítem' : 'Editar ítem' }}</h1>
        <p class="mt-1 text-sm text-muted-foreground">
          {{ formMode() === 'confirm' ? 'Asigna SKU interno y confirma el ítem provisional.' : 'Actualiza nombre, tipo o SKU (si aún no existe).' }}
        </p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading() && !store.selectedItem()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      }

      @if (store.selectedItem(); as item) {
        <hlm-card class="p-6">
          <app-catalog-item-form [mode]="formMode()" [initial]="item" [submitting]="store.submitting()" (submitted)="onSubmit($event)" (cancelled)="goBack()" />
        </hlm-card>
      }
    </div>
  `,
})
export class EditItemPage implements OnInit {
  readonly store = inject(CatalogStore);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly formMode = computed<CatalogItemFormMode>(() => {
    const item = this.store.selectedItem();
    return item?.status === 'provisional' ? 'confirm' : 'edit';
  });

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('itemId');
    if (id) this.store.loadItem(id);
  }

  onSubmit(value: CatalogItemFormValue): void {
    const item = this.store.selectedItem();
    if (!item) return;

    const input: UpdateCatalogItemInput = {
      name: value.name,
      kind: value.kind,
    };
    if (!item.internal_sku && value.internal_sku) {
      input.internal_sku = value.internal_sku;
    }
    if (this.formMode() === 'confirm') {
      input.status = 'confirmed';
    }

    this.store.updateItem(item.id, input).subscribe((updated) => {
      if (updated) void this.router.navigate(['..'], { relativeTo: this.route });
    });
  }

  goBack(): void {
    void this.router.navigate(['..'], { relativeTo: this.route });
  }
}

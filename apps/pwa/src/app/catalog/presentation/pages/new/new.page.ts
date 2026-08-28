import { Component, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { CatalogStore } from '../../../application/catalog.store';
import { generateUlid } from '../../../../core/utils/ulid';
import { CatalogItemFormComponent, CatalogItemFormValue } from '../../components/catalog-item-form/catalog-item-form.component';

@Component({
  selector: 'app-catalog-item-new',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmCardImports, HlmAlertImports, CatalogItemFormComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al catálogo
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Nuevo ítem</h1>
        <p class="mt-1 text-sm text-muted-foreground">Crea un ítem confirmado con SKU interno obligatorio.</p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      <hlm-card class="p-6">
        <app-catalog-item-form mode="create" [submitting]="store.submitting()" (submitted)="onSubmit($event)" (cancelled)="goBack()" />
      </hlm-card>
    </div>
  `,
})
export class NewItemPage {
  readonly store = inject(CatalogStore);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  onSubmit(value: CatalogItemFormValue): void {
    this.store
      .createItem({
        id: generateUlid(),
        name: value.name,
        kind: value.kind,
        internal_sku: value.internal_sku,
      })
      .subscribe((item) => {
        if (item) void this.router.navigate(['..', item.id], { relativeTo: this.route });
      });
  }

  goBack(): void {
    void this.router.navigate(['..'], { relativeTo: this.route });
  }
}

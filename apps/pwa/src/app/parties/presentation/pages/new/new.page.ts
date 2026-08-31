import { Component, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { PartiesStore } from '../../../application/parties.store';
import { PartyFormComponent, PartyFormValue } from '../../components/party-form/party-form.component';

@Component({
  selector: 'app-parties-new',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmCardImports, HlmAlertImports, PartyFormComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver a contactos
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Nuevo contacto</h1>
        <p class="mt-1 text-sm text-muted-foreground">Registra un proveedor o cliente con NIT.</p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      <hlm-card class="p-6">
        <app-party-form mode="create" [submitting]="store.submitting()" (submitted)="onSubmit($event)" (cancelled)="goBack()" />
      </hlm-card>
    </div>
  `,
})
export class NewPartyPage {
  readonly store = inject(PartiesStore);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);

  onSubmit(value: PartyFormValue): void {
    this.store.createParty({ name: value.name, tax_id: value.tax_id, roles: value.roles }).subscribe((party) => {
      if (party) void this.router.navigate(['..', party.id], { relativeTo: this.route });
    });
  }

  goBack(): void {
    void this.router.navigate(['..'], { relativeTo: this.route });
  }
}

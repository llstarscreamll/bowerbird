import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { PartiesStore } from '../../../application/parties.store';
import { PartyFormComponent, PartyFormValue } from '../../components/party-form/party-form.component';

@Component({
  selector: 'app-parties-edit',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmCardImports, HlmAlertImports, HlmSpinnerImports, PartyFormComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al detalle
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Editar contacto</h1>
        <p class="mt-1 text-sm text-muted-foreground">Actualiza nombre o roles. El NIT no se puede modificar.</p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      @if (store.loading() && !store.selectedParty()) {
        <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
      }

      @if (store.selectedParty(); as party) {
        <hlm-card class="p-6">
          <app-party-form mode="edit" [initial]="party" [submitting]="store.submitting()" (submitted)="onSubmit($event)" (cancelled)="goBack()" />
        </hlm-card>
      }
    </div>
  `,
})
export class EditPartyPage implements OnInit {
  readonly store = inject(PartiesStore);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('partyId');
    if (id) this.store.loadParty(id);
  }

  onSubmit(value: PartyFormValue): void {
    const party = this.store.selectedParty();
    if (!party) return;

    this.store.updateParty(party.id, { name: value.name, roles: value.roles }).subscribe((updated) => {
      if (updated) void this.router.navigate(['..'], { relativeTo: this.route });
    });
  }

  goBack(): void {
    void this.router.navigate(['..'], { relativeTo: this.route });
  }
}

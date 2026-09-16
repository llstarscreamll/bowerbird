import { Component, OnInit, inject } from '@angular/core';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { PartiesStore } from '../../../application/parties.store';
import { creationSourceLabel } from '../../../domain/party.model';
import { PartyFormComponent, PartyFormValue } from '../../components/party-form/party-form.component';

@Component({
  selector: 'app-parties-edit',
  standalone: true,
  imports: [RouterLink, NgIcon, HlmCardImports, HlmAlertImports, HlmSpinnerImports, HlmButtonImports, HlmBadgeImports, PartyFormComponent],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-lg space-y-6">
      <a routerLink=".." class="inline-flex items-center text-sm text-muted-foreground hover:text-foreground">
        <ng-icon name="lucideArrowLeft" class="mr-1" />
        Volver al detalle
      </a>
      <header>
        <h1 class="text-2xl font-semibold tracking-tight">Editar contacto</h1>
        <p class="mt-1 text-sm text-muted-foreground">Actualiza identidad y canales. El número de documento no se puede modificar.</p>
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

        <hlm-card class="space-y-3 p-6">
          <h2 class="text-sm font-medium">Correos</h2>
          @for (email of party.emails; track email.id) {
            <div class="flex items-center justify-between gap-2 text-sm">
              <span
                >{{ email.value }} <span hlmBadge variant="outline">{{ creationSourceLabel(email.source) }}</span></span
              >
              <button hlmBtn variant="ghost" size="sm" type="button" (click)="removeEmail(party.id, email.id)">Quitar</button>
            </div>
          }
          <div class="flex gap-2">
            <input
              class="flex h-10 flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
              [value]="emailDraft"
              (input)="emailDraft = $any($event.target).value"
              placeholder="correo@empresa.com"
            />
            <button hlmBtn type="button" variant="outline" [disabled]="!emailDraft.trim() || store.submitting()" (click)="addEmail(party.id)">Añadir</button>
          </div>
        </hlm-card>

        <hlm-card class="space-y-3 p-6">
          <h2 class="text-sm font-medium">Teléfonos</h2>
          @for (phone of party.phones; track phone.id) {
            <div class="flex items-center justify-between gap-2 text-sm">
              <span
                >{{ phone.value }} <span hlmBadge variant="outline">{{ creationSourceLabel(phone.source) }}</span></span
              >
              <button hlmBtn variant="ghost" size="sm" type="button" (click)="removePhone(party.id, phone.id)">Quitar</button>
            </div>
          }
          <div class="flex gap-2">
            <input
              class="flex h-10 flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
              [value]="phoneDraft"
              (input)="phoneDraft = $any($event.target).value"
              placeholder="Teléfono"
            />
            <button hlmBtn type="button" variant="outline" [disabled]="!phoneDraft.trim() || store.submitting()" (click)="addPhone(party.id)">Añadir</button>
          </div>
        </hlm-card>

        <hlm-card class="space-y-3 p-6">
          <h2 class="text-sm font-medium">Direcciones</h2>
          @for (addr of party.addresses; track addr.id) {
            <div class="flex items-start justify-between gap-2 text-sm">
              <span>{{ addr.line }}{{ addr.city ? ', ' + addr.city : '' }}</span>
              <button hlmBtn variant="ghost" size="sm" type="button" (click)="removeAddress(party.id, addr.id)">Quitar</button>
            </div>
          }
          <div class="grid gap-2">
            <input class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm" [value]="addrLine" (input)="addrLine = $any($event.target).value" placeholder="Dirección" />
            <div class="flex gap-2">
              <input class="flex h-10 flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm" [value]="addrCity" (input)="addrCity = $any($event.target).value" placeholder="Ciudad" />
              <input
                class="flex h-10 flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm"
                [value]="addrDept"
                (input)="addrDept = $any($event.target).value"
                placeholder="Departamento"
              />
            </div>
            <button hlmBtn type="button" variant="outline" [disabled]="!(addrLine.trim() || addrCity.trim()) || store.submitting()" (click)="addAddress(party.id)">Añadir dirección</button>
          </div>
        </hlm-card>
      }
    </div>
  `,
})
export class EditPartyPage implements OnInit {
  readonly store = inject(PartiesStore);
  readonly creationSourceLabel = creationSourceLabel;
  emailDraft = '';
  phoneDraft = '';
  addrLine = '';
  addrCity = '';
  addrDept = '';
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  ngOnInit(): void {
    const id = this.route.snapshot.paramMap.get('partyId');
    if (id) this.store.loadParty(id);
  }

  onSubmit(value: PartyFormValue): void {
    const party = this.store.selectedParty();
    if (!party) return;

    this.store.updateParty(party.id, { name: value.name, roles: value.roles, scheme_id: value.scheme_id || undefined }).subscribe((updated) => {
      if (updated) void this.router.navigate(['..'], { relativeTo: this.route });
    });
  }

  addEmail(id: string): void {
    const value = this.emailDraft.trim();
    if (!value) return;
    this.store.addEmail(id, value).subscribe((party) => {
      if (party) this.emailDraft = '';
    });
  }

  addPhone(id: string): void {
    const value = this.phoneDraft.trim();
    if (!value) return;
    this.store.addPhone(id, value).subscribe((party) => {
      if (party) this.phoneDraft = '';
    });
  }

  addAddress(id: string): void {
    this.store
      .addAddress(id, {
        line: this.addrLine.trim(),
        city: this.addrCity.trim(),
        department: this.addrDept.trim(),
        postal_zone: '',
        country_code: 'CO',
        kind: 'other',
      })
      .subscribe((party) => {
        if (party) {
          this.addrLine = '';
          this.addrCity = '';
          this.addrDept = '';
        }
      });
  }

  removeEmail(id: string, emailId: string): void {
    this.store.deleteEmail(id, emailId).subscribe();
  }

  removePhone(id: string, phoneId: string): void {
    this.store.deletePhone(id, phoneId).subscribe();
  }

  removeAddress(id: string, addressId: string): void {
    this.store.deleteAddress(id, addressId).subscribe();
  }

  goBack(): void {
    void this.router.navigate(['..'], { relativeTo: this.route });
  }
}

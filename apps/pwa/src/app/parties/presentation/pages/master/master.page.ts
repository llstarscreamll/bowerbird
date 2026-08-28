import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { HlmAlertImports } from '@spartan-ng/helm/alert';
import { HlmBadgeImports } from '@spartan-ng/helm/badge';
import { HlmCardImports } from '@spartan-ng/helm/card';
import { HlmSpinnerImports } from '@spartan-ng/helm/spinner';
import { HlmTableImports } from '@spartan-ng/helm/table';
import { NgIcon } from '@ng-icons/core';
import { PartiesStore } from '../../../application/parties.store';

@Component({
  selector: 'app-parties-master',
  standalone: true,
  imports: [CommonModule, NgIcon, HlmCardImports, HlmSpinnerImports, HlmTableImports, HlmAlertImports, HlmBadgeImports],
  host: { class: 'flex-1 flex flex-col min-h-0 w-full overflow-y-auto p-8' },
  template: `
    <div class="mx-auto w-full max-w-5xl space-y-6">
      <header>
        <h1 class="text-2xl font-semibold tracking-tight sm:text-3xl">Contactos</h1>
        <p class="mt-1 text-sm text-muted-foreground">Proveedores y clientes identificados por NIT, creados desde facturas electrónicas.</p>
      </header>

      @if (store.errorMessage(); as err) {
        <div hlmAlert variant="destructive">
          <ng-icon name="lucideCircleAlert" hlmAlertIcon />
          <h4 hlmAlertTitle>Error</h4>
          <p hlmAlertDescription>{{ err }}</p>
        </div>
      }

      <hlm-card class="overflow-hidden p-0">
        @if (store.loading()) {
          <div class="flex justify-center py-16"><hlm-spinner class="size-8" /></div>
        } @else {
          <table hlmTable>
            <thead hlmTHead>
              <tr hlmTr>
                <th hlmTh>Nombre</th>
                <th hlmTh>NIT</th>
                <th hlmTh>Roles</th>
                <th hlmTh>Estado</th>
              </tr>
            </thead>
            <tbody hlmTBody>
              @for (party of store.parties(); track party.id) {
                <tr hlmTr>
                  <td hlmTd class="font-medium">{{ party.name }}</td>
                  <td hlmTd class="text-muted-foreground">{{ party.tax_id || '—' }}</td>
                  <td hlmTd>
                    @for (role of party.roles; track role) {
                      <span hlmBadge variant="secondary" class="mr-1">{{ role }}</span>
                    }
                  </td>
                  <td hlmTd>{{ party.status }}</td>
                </tr>
              } @empty {
                <tr hlmTr>
                  <td hlmTd colspan="4" class="py-10 text-center text-muted-foreground">Aún no hay contactos.</td>
                </tr>
              }
            </tbody>
          </table>
        }
      </hlm-card>
    </div>
  `,
})
export class MasterPage implements OnInit {
  readonly store = inject(PartiesStore);

  ngOnInit(): void {
    this.store.load();
  }
}

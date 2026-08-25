import { CommonModule } from '@angular/common';
import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive, ActivatedRoute, Router, NavigationEnd } from '@angular/router';
import { NgIcon } from '@ng-icons/core';
import { filter } from 'rxjs';
import { TenantHttpService } from '../../../../tenant/infrastructure/tenant.http.service';
import { AuthStore } from '../../../../auth/application/auth.store';
import { TenantContextStore } from '../../../store/tenant-context.store';
import { EntitlementsStore } from '../../../../entitlements/application/entitlements.store';
import { PermissionsStore } from '../../../../rbac/application/permissions.store';
import { HlmAvatarImports } from '@spartan-ng/helm/avatar';
import { HlmButtonImports } from '@spartan-ng/helm/button';
import { HlmDropdownMenuImports } from '@spartan-ng/helm/dropdown-menu';
import { HlmSeparatorImports } from '@spartan-ng/helm/separator';
import { HlmSidebarImports, HlmSidebarService } from '@spartan-ng/helm/sidebar';
import { HlmTooltipImports } from '@spartan-ng/helm/tooltip';

@Component({
  selector: 'app-tenant-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive, NgIcon, HlmSidebarImports, HlmDropdownMenuImports, HlmAvatarImports, HlmButtonImports, HlmSeparatorImports, HlmTooltipImports],
  template: `
    <div hlmSidebarWrapper class="h-screen w-full overflow-hidden">
      <hlm-sidebar collapsible="icon" side="left">
        <hlm-sidebar-header class="border-b border-sidebar-border">
          <div class="flex items-center gap-3 px-2 py-2 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-0">
            <div class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
              <ng-icon name="lucideBird" class="text-lg" />
            </div>
            <span class="truncate font-bold tracking-tight group-data-[collapsible=icon]:hidden">Bowerbird</span>
          </div>
        </hlm-sidebar-header>

        <hlm-sidebar-content>
          <hlm-sidebar-group class="group-data-[collapsible=icon]:hidden">
            <button hlmBtn variant="outline" class="h-auto w-full min-w-0 justify-between gap-2 overflow-hidden px-2.5 py-2" [hlmDropdownMenuTrigger]="tenantMenu">
              <div class="flex min-w-0 flex-1 items-center gap-2.5">
                <div class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary">
                  <ng-icon name="lucideBuilding2" class="text-base" />
                </div>
                <div class="min-w-0 flex-1 text-left leading-tight">
                  <div class="truncate text-sm font-semibold [text-box:trim-both_cap_alphabetic]">{{ tenantName() }}</div>
                  <div class="mt-0.5 flex min-w-0 items-center gap-1 text-xs text-muted-foreground">
                    <ng-icon name="lucideUsers" class="shrink-0 text-xs" />
                    <span class="truncate [text-box:trim-both_cap_alphabetic]">{{ tenantMembers() }} Miembro{{ tenantMembers() !== 1 ? 's' : '' }}</span>
                  </div>
                </div>
              </div>
              <ng-icon name="lucideChevronsUpDown" class="size-4 shrink-0 text-muted-foreground" />
            </button>
            <ng-template #tenantMenu>
              <hlm-dropdown-menu class="w-56">
                <a hlmDropdownMenuItem routerLink="/lobby" (click)="closeTenantMenu()">
                  <ng-icon name="lucideList" />
                  Todas las organizaciones
                </a>
                <hlm-separator class="my-1" />
                <a hlmDropdownMenuItem routerLink="/lobby" [queryParams]="{ create: true }" (click)="closeTenantMenu()">
                  <ng-icon name="lucideBuilding" />
                  Crear nueva organización
                </a>
              </hlm-dropdown-menu>
            </ng-template>
          </hlm-sidebar-group>

          <hlm-sidebar-group>
            <div hlmSidebarGroupLabel class="group-data-[collapsible=icon]:hidden">Navegación</div>
            <ul hlmSidebarMenu>
              <li hlmSidebarMenuItem>
                <a hlmSidebarMenuButton [routerLink]="['/', tenantId(), 'dashboard']" routerLinkActive #dashboardLink="routerLinkActive" [isActive]="dashboardLink.isActive" [tooltip]="'Dashboard'">
                  <ng-icon name="lucideLayoutDashboard" />
                  <span>Dashboard</span>
                </a>
              </li>
              @if (entitlements.hasMailInbox()) {
                <li hlmSidebarMenuItem>
                  <a
                    hlmSidebarMenuButton
                    [routerLink]="['/', tenantId(), 'inbox', 'master']"
                    routerLinkActive
                    #inboxLink="routerLinkActive"
                    [routerLinkActiveOptions]="{ exact: false }"
                    [isActive]="inboxLink.isActive"
                    [tooltip]="'Mails'"
                  >
                    <ng-icon name="lucideInbox" />
                    <span>Mails</span>
                  </a>
                </li>
              }
              @if (entitlements.hasInvoicing()) {
                <li hlmSidebarMenuItem>
                  <a hlmSidebarMenuButton [routerLink]="['/', tenantId(), 'invoices']" routerLinkActive #invoicesLink="routerLinkActive" [isActive]="invoicesLink.isActive" [tooltip]="'Facturas'">
                    <ng-icon name="lucideReceipt" />
                    <span>Facturas</span>
                  </a>
                </li>
                <li hlmSidebarMenuItem>
                  <a hlmSidebarMenuButton [routerLink]="['/', tenantId(), 'parties']" routerLinkActive #partiesLink="routerLinkActive" [isActive]="partiesLink.isActive" [tooltip]="'Contrapartes'">
                    <ng-icon name="lucideBuilding2" />
                    <span>Contrapartes</span>
                  </a>
                </li>
                <li hlmSidebarMenuItem>
                  <a
                    hlmSidebarMenuButton
                    [routerLink]="['/', tenantId(), 'catalog']"
                    routerLinkActive
                    #catalogLink="routerLinkActive"
                    [routerLinkActiveOptions]="{ exact: false }"
                    [isActive]="catalogLink.isActive"
                    [tooltip]="'Catálogo'"
                  >
                    <ng-icon name="lucidePackage" />
                    <span>Catálogo</span>
                  </a>
                </li>
              }
              @if (entitlements.showConnections()) {
                <li hlmSidebarMenuItem>
                  <a
                    hlmSidebarMenuButton
                    [routerLink]="['/', tenantId(), 'connections']"
                    routerLinkActive
                    #connectionsLink="routerLinkActive"
                    [routerLinkActiveOptions]="{ exact: false }"
                    [isActive]="connectionsLink.isActive"
                    [tooltip]="'Cuentas asociadas'"
                  >
                    <ng-icon name="lucideLink" />
                    <span>Cuentas</span>
                  </a>
                </li>
              }
              @if (permissions.canReadSecrets()) {
                <li hlmSidebarMenuItem>
                  <a hlmSidebarMenuButton [routerLink]="['/', tenantId(), 'secrets']" routerLinkActive #secretsLink="routerLinkActive" [isActive]="secretsLink.isActive" [tooltip]="'Credenciales'">
                    <ng-icon name="lucideKeyRound" />
                    <span>Credenciales</span>
                  </a>
                </li>
              }
            </ul>
          </hlm-sidebar-group>
        </hlm-sidebar-content>

        <hlm-sidebar-footer class="border-t border-sidebar-border">
          <button hlmBtn variant="ghost" class="mb-2 w-full justify-between group-data-[collapsible=icon]:px-2" (click)="sidebarService.toggleSidebar()">
            <span class="group-data-[collapsible=icon]:hidden">Contraer menú</span>
            <ng-icon [name]="sidebarService.state() === 'collapsed' ? 'lucideChevronsRight' : 'lucideChevronsLeft'" />
          </button>

          <button hlmBtn variant="ghost" class="w-full justify-start gap-3 px-2" [hlmDropdownMenuTrigger]="userMenu">
            <hlm-avatar size="sm">
              @if (userAvatar(); as avatar) {
                <img hlmAvatarImage [src]="avatar" alt="User avatar" />
              }
              <span hlmAvatarFallback>{{ userInitials() }}</span>
            </hlm-avatar>
            <div class="min-w-0 flex-1 text-left group-data-[collapsible=icon]:hidden">
              <div class="truncate text-sm font-medium">{{ userName() }}</div>
              <div class="truncate text-xs capitalize text-muted-foreground">{{ translatedRole() }}</div>
            </div>
            <ng-icon name="lucideChevronsUpDown" class="text-muted-foreground group-data-[collapsible=icon]:hidden" />
          </button>
          <ng-template #userMenu>
            <hlm-dropdown-menu class="w-56">
              <div class="flex items-center justify-between px-2 py-1.5">
                <span class="text-sm font-medium">Tema</span>
                <div class="flex items-center rounded-full border bg-muted p-1">
                  <button type="button" hlmBtn size="icon-sm" [variant]="themeMode() === 'system' ? 'secondary' : 'ghost'" (click)="setTheme('system')" title="Sistema">
                    <ng-icon name="lucideMonitor" />
                  </button>
                  <button type="button" hlmBtn size="icon-sm" [variant]="themeMode() === 'light' ? 'secondary' : 'ghost'" (click)="setTheme('light')" title="Claro">
                    <ng-icon name="lucideSun" />
                  </button>
                  <button type="button" hlmBtn size="icon-sm" [variant]="themeMode() === 'dark' ? 'secondary' : 'ghost'" (click)="setTheme('dark')" title="Oscuro">
                    <ng-icon name="lucideMoon" />
                  </button>
                </div>
              </div>
              <hlm-separator class="my-1" />
              <button hlmDropdownMenuItem variant="destructive" (click)="logout()">
                <ng-icon name="lucideLogOut" />
                Cerrar sesión
              </button>
            </hlm-dropdown-menu>
          </ng-template>
        </hlm-sidebar-footer>
        <button hlmSidebarRail type="button" class="sr-only" aria-hidden="true"></button>
      </hlm-sidebar>

      <main hlmSidebarInset class="min-h-0 overflow-hidden bg-muted/30">
        <router-outlet />
      </main>
    </div>
  `,
})
export class TenantLayoutComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private tenantService = inject(TenantHttpService);
  private authStore = inject(AuthStore);
  private tenantContextStore = inject(TenantContextStore);
  readonly entitlements = inject(EntitlementsStore);
  readonly permissions = inject(PermissionsStore);
  readonly sidebarService = inject(HlmSidebarService);

  themeMode = signal<'system' | 'light' | 'dark'>('system');

  tenantId = this.tenantContextStore.tenantId;
  tenantDetails = this.tenantContextStore.tenantDetails;

  private get decodedToken(): any {
    const token = this.authStore.accessToken();
    if (!token) return null;
    try {
      const payload = token.split('.')[1];
      return JSON.parse(atob(payload));
    } catch {
      return null;
    }
  }

  userName = computed(() => {
    const claims = this.decodedToken;
    if (!claims) return 'Usuario';
    return `${claims.first_name || ''} ${claims.last_name || ''}`.trim() || 'Usuario';
  });

  userInitials = computed(() => {
    const claims = this.decodedToken;
    if (!claims) return 'U';
    return (claims.first_name?.[0] || '') + (claims.last_name?.[0] || '') || 'U';
  });

  userAvatar = computed(() => this.decodedToken?.picture_url || null);

  translatedRole = computed(() => {
    const role = this.tenantDetails()?.current_user_role;
    switch (role) {
      case 'owner':
        return 'Propietario';
      case 'admin':
        return 'Administrador';
      case 'member':
        return 'Miembro';
      default:
        return role || 'Miembro';
    }
  });

  tenantName = computed(() => {
    const details = this.tenantDetails();
    if (details) return details.name;

    const id = this.tenantId();
    if (!id) return 'Cargando...';
    return id
      .split('-')
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  });

  tenantMembers = computed(() => {
    const details = this.tenantDetails();
    return details?.members_count ?? 0;
  });

  ngOnInit() {
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null;
    if (savedTheme) {
      this.themeMode.set(savedTheme);
      document.documentElement.classList.toggle('dark', savedTheme === 'dark');
    } else {
      this.themeMode.set('system');
      document.documentElement.classList.toggle('dark', window.matchMedia('(prefers-color-scheme: dark)').matches);
    }

    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (!localStorage.getItem('theme')) {
        document.documentElement.classList.toggle('dark', e.matches);
      }
    });

    this.updateTenantId();

    this.router.events.pipe(filter((event) => event instanceof NavigationEnd)).subscribe(() => {
      this.updateTenantId();
    });
  }

  closeTenantMenu() {}

  logout() {
    this.authStore.logout({
      onFinish: () => this.router.navigate(['/login']),
    });
  }

  setTheme(mode: 'system' | 'light' | 'dark') {
    this.themeMode.set(mode);
    if (mode === 'system') {
      localStorage.removeItem('theme');
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.classList.toggle('dark', isDark);
    } else {
      localStorage.setItem('theme', mode);
      document.documentElement.classList.toggle('dark', mode === 'dark');
    }
  }

  private updateTenantId() {
    let currentRoute: ActivatedRoute | null = this.route;
    let newTenantId = '';

    while (currentRoute) {
      if (currentRoute.snapshot.paramMap.has('tenantId')) {
        newTenantId = currentRoute.snapshot.paramMap.get('tenantId')!;
        break;
      }
      currentRoute = currentRoute.firstChild;
    }

    if (newTenantId && newTenantId !== this.tenantId()) {
      this.tenantContextStore.setTenantId(newTenantId);
      this.entitlements.load(newTenantId).subscribe();
      this.permissions.load(newTenantId).subscribe();
    } else if (newTenantId && this.entitlements.loadedTenantId() !== newTenantId) {
      this.entitlements.load(newTenantId).subscribe();
      this.permissions.load(newTenantId).subscribe();
    } else if (newTenantId && this.permissions.loadedTenantId() !== newTenantId) {
      this.permissions.load(newTenantId).subscribe();
    }
  }
}

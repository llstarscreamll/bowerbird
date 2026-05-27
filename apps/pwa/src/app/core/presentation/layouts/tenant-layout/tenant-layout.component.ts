import { CommonModule } from '@angular/common';
import { Component, OnInit, signal, computed, inject } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive, ActivatedRoute, Router, NavigationEnd } from '@angular/router';
import { filter } from 'rxjs';
import { OrganizationHttpService, OrganizationResponse } from '../../../../organization/infrastructure/organization.http.service';
import { AuthStore } from '../../../../auth/application/auth.store';

@Component({
  selector: 'app-tenant-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    <div class="flex h-screen w-full bg-white dark:bg-slate-900 text-slate-900 dark:text-white font-sans overflow-hidden transition-colors duration-200">
      <!-- Sidebar -->
      <aside
        class="flex flex-col bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 transition-all duration-300 ease-in-out z-20 relative"
        [class.w-[260px]]="!isCollapsed()"
        [class.w-[80px]]="isCollapsed()"
      >
        <!-- Logo / Header -->
        <div class="flex items-center h-16 px-6">
          <div class="flex items-center gap-3 w-full" [class.justify-center]="isCollapsed()">
            <div class="flex items-center justify-center w-8 h-8 rounded-lg bg-slate-900 dark:bg-slate-800 text-white shrink-0 shadow-sm">
              <span class="material-icons-outlined text-xl">bubble_chart</span>
            </div>
            <span
              class="font-bold text-lg tracking-tight whitespace-nowrap transition-opacity duration-200 text-slate-900 dark:text-white"
              [class.opacity-0]="isCollapsed()"
              [class.w-0]="isCollapsed()"
              [class.hidden]="isCollapsed()"
            >
              Bowerbird
            </span>
          </div>
        </div>

        <!-- Tenant Selector -->
        <div class="px-4 py-3 relative" [class.hidden]="isCollapsed()">
          <!-- Backdrop -->
          <div *ngIf="isTenantMenuOpen()" (click)="closeTenantMenu()" class="fixed inset-0 z-40"></div>

          <button
            (click)="toggleTenantMenu()"
            class="w-full flex items-center justify-between p-2.5 rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-[0_1px_2px_rgba(0,0,0,0.02)] hover:bg-slate-50 dark:hover:bg-slate-800 hover:border-slate-300 dark:hover:border-slate-600 transition-all group relative z-50"
          >
            <div class="flex items-center gap-3 min-w-0">
              <!-- Icon -->
              <div class="w-10 h-10 rounded-lg bg-indigo-100 dark:bg-indigo-900/40 text-indigo-600 dark:text-indigo-400 flex items-center justify-center shrink-0">
                <span class="material-icons-outlined text-[20px]">apartment</span>
              </div>
              <!-- Text Info -->
              <div class="flex flex-col items-start overflow-hidden text-left min-w-0">
                <span class="text-sm font-semibold text-slate-900 dark:text-white truncate w-full">{{ tenantName() }}</span>
                <span class="text-xs text-slate-500 dark:text-slate-400 flex items-center gap-1 mt-0.5">
                  <span class="material-icons-outlined text-sm">group</span>
                  {{ tenantMembers() }} Miembro{{ tenantMembers() !== 1 ? 's' : '' }}
                </span>
              </div>
            </div>
            <span
              class="material-icons-outlined text-slate-400 dark:text-slate-500 text-[18px] shrink-0 group-hover:text-slate-600 dark:group-hover:text-slate-300 transition-transform duration-200"
              [class.rotate-180]="isTenantMenuOpen()"
              >unfold_more</span
            >
          </button>

          <!-- Dropdown -->
          <div
            *ngIf="isTenantMenuOpen()"
            class="absolute left-4 right-4 top-[calc(100%-4px)] bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl shadow-lg z-50 py-1.5 overflow-hidden animate-in fade-in zoom-in-95 duration-100"
          >
            <a
              routerLink="/lobby"
              (click)="closeTenantMenu()"
              class="flex items-center gap-2.5 px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/50 hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              <span class="material-icons-outlined text-[18px] text-slate-400 dark:text-slate-500">format_list_bulleted</span>
              Todas las organizaciones
            </a>
            <div class="h-px bg-slate-100 dark:bg-slate-700 my-1 mx-3"></div>
            <a
              routerLink="/lobby"
              [queryParams]="{ create: true }"
              (click)="closeTenantMenu()"
              class="flex items-center gap-2.5 px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/50 hover:text-slate-900 dark:hover:text-white transition-colors"
            >
              <span class="material-icons-outlined text-[18px] text-slate-400 dark:text-slate-500">add_business</span>
              Crear nueva organización
            </a>
          </div>
        </div>

        <!-- Divider -->
        <div class="px-4 pb-2" [class.hidden]="isCollapsed()">
          <div class="h-px w-full bg-slate-100 dark:bg-slate-800 transition-colors"></div>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 overflow-y-auto py-2 px-4 space-y-2 text-slate-600 dark:text-slate-400">
          <!-- Dashboard Link -->
          <a
            [routerLink]="['/', tenantId(), 'dashboard']"
            routerLinkActive="bg-indigo-50 dark:bg-indigo-500/10 text-indigo-800 dark:text-indigo-100 font-medium"
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors group cursor-pointer"
            [class.justify-center]="isCollapsed()"
            [title]="isCollapsed() ? 'Dashboard' : ''"
          >
            <span class="material-icons-outlined text-lg group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors" routerLinkActive="text-indigo-600 dark:text-indigo-400"
              >space_dashboard</span
            >
            <span class="whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()"> Dashboard </span>
          </a>

          <!-- Inbox Link -->
          <a
            [routerLink]="['/', tenantId(), 'inbox', 'unified']"
            routerLinkActive="bg-indigo-50 dark:bg-indigo-500/10 text-indigo-800 dark:text-indigo-100 font-medium"
            [routerLinkActiveOptions]="{ exact: false }"
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors group cursor-pointer"
            [class.justify-center]="isCollapsed()"
            [title]="isCollapsed() ? 'Inbox' : ''"
          >
            <span class="material-icons-outlined text-lg group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors" routerLinkActive="text-indigo-600 dark:text-indigo-400"
              >inbox</span
            >
            <span class="whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()"> Mails </span>
          </a>

          <!-- Invoices Link -->
          <a
            [routerLink]="['/', tenantId(), 'invoices']"
            routerLinkActive="bg-indigo-50 dark:bg-indigo-500/10 text-indigo-800 dark:text-indigo-100 font-medium"
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors group cursor-pointer"
            [class.justify-center]="isCollapsed()"
            [title]="isCollapsed() ? 'Facturas' : ''"
          >
            <span class="material-icons-outlined text-lg group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors" routerLinkActive="text-indigo-600 dark:text-indigo-400"
              >receipt_long</span
            >
            <span class="whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()"> Facturas </span>
          </a>
        </nav>

        <!-- Toggle Button -->
        <div class="px-4 py-2 border-t border-slate-100 dark:border-slate-800 flex justify-end transition-colors">
          <button
            (click)="toggleSidebar()"
            class="flex items-center gap-4 h-8 w-full px-4 rounded-lg text-slate-400 dark:text-slate-500 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            [title]="isCollapsed() ? 'Expandir' : 'Contraer'"
          >
            <span class="whitespace-nowrap transition-opacity duration-200 grow text-left text-slate-600 dark:text-slate-300" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()">
              Contraer menú
            </span>
            <span class="material-icons-outlined text-sm">
              {{ isCollapsed() ? 'keyboard_double_arrow_right' : 'keyboard_double_arrow_left' }}
            </span>
          </button>
        </div>

        <!-- User Menu (Bottom) -->
        <div class="p-4 border-t border-slate-100 dark:border-slate-800 transition-colors relative">
          <!-- Backdrop -->
          <div *ngIf="isUserMenuOpen()" (click)="closeUserMenu()" class="fixed inset-0 z-40"></div>

          <button
            (click)="toggleUserMenu()"
            class="flex items-center gap-3 w-full hover:bg-slate-50 dark:hover:bg-slate-800 p-2 rounded-lg transition-colors group relative z-50"
            [class.justify-center]="isCollapsed()"
          >
            <div
              class="flex items-center justify-center w-8 h-8 rounded-full bg-slate-200 dark:bg-slate-700 text-slate-600 dark:text-slate-300 shrink-0 font-medium text-sm transition-colors overflow-hidden"
            >
              <img *ngIf="userAvatar(); else fallbackAvatar" [src]="userAvatar()" alt="User avatar" class="w-full h-full object-cover" />
              <ng-template #fallbackAvatar>{{ userInitials() }}</ng-template>
            </div>
            <div class="flex flex-col items-start overflow-hidden whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()">
              <span class="text-sm font-medium text-slate-900 dark:text-white truncate w-full text-left transition-colors">{{ userName() }}</span>
              <span class="text-xs text-slate-500 dark:text-slate-400 truncate w-full text-left transition-colors capitalize">{{ translatedRole() }}</span>
            </div>
            <span
              class="material-icons-outlined text-slate-400 dark:text-slate-500 group-hover:text-slate-600 dark:group-hover:text-slate-300 text-sm ml-auto transition-all duration-200"
              [class.hidden]="isCollapsed()"
              [class.rotate-180]="isUserMenuOpen()"
              >unfold_more</span
            >
          </button>

          <!-- Dropdown -->
          <div
            *ngIf="isUserMenuOpen()"
            class="absolute left-4 right-4 bottom-[calc(100%-4px)] bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl shadow-lg z-50 py-1.5 overflow-hidden animate-in fade-in slide-in-from-bottom-2 duration-100"
          >
            <!-- Theme Toggle Segmented Control -->
            <div class="px-3 py-2 flex items-center justify-between">
              <span class="text-sm font-medium text-slate-700 dark:text-slate-300">Tema</span>
              <div class="flex items-center bg-slate-100 dark:bg-slate-900 rounded-full p-1 border border-slate-200 dark:border-slate-700/50 shadow-[inset_0_1px_2px_rgba(0,0,0,0.05)]">
                <!-- System -->
                <button
                  (click)="setTheme('system')"
                  class="w-7 h-7 rounded-full flex items-center justify-center transition-all duration-200"
                  [class.bg-white]="themeMode() === 'system'"
                  [class.dark:bg-slate-700]="themeMode() === 'system'"
                  [class.shadow-sm]="themeMode() === 'system'"
                  [class.text-slate-900]="themeMode() === 'system'"
                  [class.dark:text-white]="themeMode() === 'system'"
                  [class.text-slate-400]="themeMode() !== 'system'"
                  [class.hover:text-slate-600]="themeMode() !== 'system'"
                  [class.dark:hover:text-slate-300]="themeMode() !== 'system'"
                  title="Sistema"
                >
                  <span class="material-icons-outlined text-[16px]">desktop_windows</span>
                </button>
                <!-- Light -->
                <button
                  (click)="setTheme('light')"
                  class="w-7 h-7 rounded-full flex items-center justify-center transition-all duration-200 ml-0.5"
                  [class.bg-white]="themeMode() === 'light'"
                  [class.dark:bg-slate-700]="themeMode() === 'light'"
                  [class.shadow-sm]="themeMode() === 'light'"
                  [class.text-slate-900]="themeMode() === 'light'"
                  [class.dark:text-white]="themeMode() === 'light'"
                  [class.text-slate-400]="themeMode() !== 'light'"
                  [class.hover:text-slate-600]="themeMode() !== 'light'"
                  [class.dark:hover:text-slate-300]="themeMode() !== 'light'"
                  title="Claro"
                >
                  <span class="material-icons-outlined text-[16px]">light_mode</span>
                </button>
                <!-- Dark -->
                <button
                  (click)="setTheme('dark')"
                  class="w-7 h-7 rounded-full flex items-center justify-center transition-all duration-200 ml-0.5"
                  [class.bg-white]="themeMode() === 'dark'"
                  [class.dark:bg-slate-700]="themeMode() === 'dark'"
                  [class.shadow-sm]="themeMode() === 'dark'"
                  [class.text-slate-900]="themeMode() === 'dark'"
                  [class.dark:text-white]="themeMode() === 'dark'"
                  [class.text-slate-400]="themeMode() !== 'dark'"
                  [class.hover:text-slate-600]="themeMode() !== 'dark'"
                  [class.dark:hover:text-slate-300]="themeMode() !== 'dark'"
                  title="Oscuro"
                >
                  <span class="material-icons-outlined text-[16px]">dark_mode</span>
                </button>
              </div>
            </div>

            <div class="h-px bg-slate-100 dark:bg-slate-700 my-1 mx-3"></div>

            <button
              (click)="logout()"
              class="w-full flex items-center gap-2.5 px-3 py-2 text-sm font-medium text-rose-600 dark:text-rose-400 hover:bg-rose-50 dark:hover:bg-rose-500/10 transition-colors"
            >
              <span class="material-icons-outlined text-[18px]">logout</span>
              Cerrar sesión
            </button>
          </div>
        </div>
      </aside>

      <!-- Main Content -->
      <main class="flex-1 flex flex-col min-w-0 overflow-hidden relative bg-slate-50 dark:bg-slate-950 transition-colors duration-200">
        <!-- Router Outlet Canvas -->
        <div class="flex-1 relative overflow-hidden flex flex-col w-full">
          <router-outlet></router-outlet>
        </div>
      </main>
    </div>
  `,
})
export class TenantLayoutComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);
  private organizationService = inject(OrganizationHttpService);
  private authStore = inject(AuthStore);

  isCollapsed = signal(false);
  isTenantMenuOpen = signal(false);
  isUserMenuOpen = signal(false);
  themeMode = signal<'system' | 'light' | 'dark'>('system');
  tenantId = signal('');
  tenantDetails = signal<OrganizationResponse | null>(null);
  isLoadingTenant = signal(false);

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
    // Formatear el tenantId (ej: acme-corp -> Acme Corp) como fallback mientras carga
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
    // Inicializar estado del modo oscuro
    const savedTheme = localStorage.getItem('theme') as 'light' | 'dark' | null;
    if (savedTheme) {
      this.themeMode.set(savedTheme);
    } else {
      this.themeMode.set('system');
    }

    // Escuchar cambios de preferencia del sistema si está en modo "system"
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (this.themeMode() === 'system') {
        document.documentElement.classList.toggle('dark', e.matches);
      }
    });

    this.updateTenantId();

    this.router.events.pipe(filter((event) => event instanceof NavigationEnd)).subscribe((event: any) => {
      this.updateTenantId();
    });
  }

  toggleSidebar() {
    this.isCollapsed.update((v) => !v);
  }

  toggleTenantMenu() {
    this.isTenantMenuOpen.update((v) => !v);
  }

  closeTenantMenu() {
    this.isTenantMenuOpen.set(false);
  }

  toggleUserMenu() {
    this.isUserMenuOpen.update((v) => !v);
  }

  closeUserMenu() {
    this.isUserMenuOpen.set(false);
  }

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
    // Traverse the route tree to find the tenantId param since it might be in children or parent
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
      this.tenantId.set(newTenantId);
      this.fetchTenantDetails(newTenantId);
    }
  }

  private fetchTenantDetails(id: string) {
    this.isLoadingTenant.set(true);
    this.organizationService.getOrganization(id).subscribe({
      next: (response) => {
        // En caso de que el backend envíe la data encapsulada en { data: ... }
        // Se asume que el interceptor/http extrae data o lo devuelve directo
        // (Ajusta la extracción según la convención json:api de tu app)
        const data = (response as any).data ? (response as any).data : response;
        this.tenantDetails.set(data);
        this.isLoadingTenant.set(false);
      },
      error: (err) => {
        console.error('Failed to load tenant details', err);
        this.isLoadingTenant.set(false);
      },
    });
  }
}

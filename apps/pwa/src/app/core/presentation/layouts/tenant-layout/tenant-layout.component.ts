import { CommonModule } from '@angular/common';
import { Component, OnInit, signal, computed, inject } from '@angular/core';
import { RouterOutlet, RouterLink, RouterLinkActive, ActivatedRoute, Router, NavigationEnd } from '@angular/router';
import { filter } from 'rxjs';

@Component({
  selector: 'app-tenant-layout',
  standalone: true,
  imports: [CommonModule, RouterOutlet, RouterLink, RouterLinkActive],
  template: `
    <div class="flex h-screen w-full bg-white text-slate-900 font-sans overflow-hidden">
      <!-- Sidebar -->
      <aside class="flex flex-col bg-white border-r border-slate-200 transition-all duration-300 ease-in-out z-20 relative" [class.w-[260px]]="!isCollapsed()" [class.w-[80px]]="isCollapsed()">
        <!-- Logo / Header -->
        <div class="flex items-center h-16 px-6">
          <div class="flex items-center gap-3 w-full" [class.justify-center]="isCollapsed()">
            <div class="flex items-center justify-center w-8 h-8 rounded-lg bg-slate-900 text-white shrink-0">
              <span class="material-icons-outlined text-xl">bubble_chart</span>
            </div>
            <span
              class="font-bold text-lg tracking-tight whitespace-nowrap transition-opacity duration-200"
              [class.opacity-0]="isCollapsed()"
              [class.w-0]="isCollapsed()"
              [class.hidden]="isCollapsed()"
            >
              Bowerbird
            </span>
          </div>
        </div>

        <!-- Tenant Selector -->
        <div class="px-4 py-3" [class.hidden]="isCollapsed()">
          <button
            class="w-full flex items-center justify-between p-2.5 rounded-xl border border-slate-200 bg-white shadow-[0_1px_2px_rgba(0,0,0,0.02)] hover:bg-slate-50 hover:border-slate-300 transition-all group"
          >
            <div class="flex items-center gap-3 min-w-0">
              <!-- Icon -->
              <div class="w-10 h-10 rounded-lg bg-indigo-100 text-indigo-600 flex items-center justify-center shrink-0">
                <span class="material-icons-outlined text-[20px]">apartment</span>
              </div>
              <!-- Text Info -->
              <div class="flex flex-col items-start overflow-hidden text-left min-w-0">
                <span class="text-sm font-semibold text-slate-900 truncate w-full">{{ tenantName() }}</span>
                <span class="text-[11px] text-slate-500 flex items-center gap-1 mt-0.5">
                  <span class="material-icons-outlined text-[13px]">group</span>
                  {{ tenantMembers() }} Miembros
                </span>
              </div>
            </div>
            <span class="material-icons-outlined text-slate-400 text-[18px] shrink-0 group-hover:text-slate-600">unfold_more</span>
          </button>
        </div>

        <!-- Divider -->
        <div class="px-4 pb-2" [class.hidden]="isCollapsed()">
          <div class="h-px w-full bg-slate-100"></div>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 overflow-y-auto py-2 px-4 space-y-2">
          <!-- Inbox Link -->
          <a
            [routerLink]="['/', tenantId(), 'inbox', 'unified']"
            routerLinkActive="bg-indigo-50 text-indigo-700 font-medium"
            [routerLinkActiveOptions]="{ exact: false }"
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-slate-600 hover:bg-slate-50 transition-colors group cursor-pointer"
            [class.justify-center]="isCollapsed()"
            [title]="isCollapsed() ? 'Inbox' : ''"
          >
            <span class="material-icons-outlined text-[22px] group-hover:text-indigo-600 transition-colors" routerLinkActive="text-indigo-600">inbox</span>
            <span class="whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()"> Bandeja Unificada </span>
          </a>

          <!-- Invoices Link (Placeholder) -->
          <a
            [routerLink]="['/', tenantId(), 'invoices']"
            routerLinkActive="bg-indigo-50 text-indigo-700 font-medium"
            class="flex items-center gap-3 px-3 py-2.5 rounded-lg text-slate-600 hover:bg-slate-50 transition-colors group cursor-pointer"
            [class.justify-center]="isCollapsed()"
            [title]="isCollapsed() ? 'Facturas' : ''"
          >
            <span class="material-icons-outlined text-[22px] group-hover:text-indigo-600 transition-colors" routerLinkActive="text-indigo-600">receipt_long</span>
            <span class="whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()"> Facturas </span>
          </a>
        </nav>

        <!-- Toggle Button -->
        <div class="px-4 py-2 border-t border-slate-100 flex justify-end">
          <button
            (click)="toggleSidebar()"
            class="flex items-center justify-center w-8 h-8 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
            [title]="isCollapsed() ? 'Expandir' : 'Contraer'"
          >
            <span class="material-icons-outlined text-sm">
              {{ isCollapsed() ? 'keyboard_double_arrow_right' : 'keyboard_double_arrow_left' }}
            </span>
          </button>
        </div>

        <!-- User Menu (Bottom) -->
        <div class="p-4 border-t border-slate-100">
          <button class="flex items-center gap-3 w-full hover:bg-slate-50 p-2 rounded-lg transition-colors" [class.justify-center]="isCollapsed()">
            <div class="flex items-center justify-center w-8 h-8 rounded-full bg-slate-200 text-slate-600 shrink-0 font-medium text-sm">U</div>
            <div class="flex flex-col items-start overflow-hidden whitespace-nowrap transition-opacity duration-200" [class.opacity-0]="isCollapsed()" [class.hidden]="isCollapsed()">
              <span class="text-sm font-medium text-slate-900 truncate w-full text-left">Usuario</span>
              <span class="text-xs text-slate-500 truncate w-full text-left">Admin</span>
            </div>
            <span class="material-icons-outlined text-slate-400 text-sm ml-auto" [class.hidden]="isCollapsed()">unfold_more</span>
          </button>
        </div>
      </aside>

      <!-- Main Content -->
      <main class="flex-1 flex flex-col min-w-0 overflow-hidden relative">
        <!-- Top Header -->
        <header class="h-16 flex items-center px-8 bg-white border-b border-slate-200 shrink-0">
          <h1 class="text-xl font-semibold text-slate-800">{{ pageTitle() }}</h1>
          <!-- Buscador eliminado como fue solicitado -->
        </header>

        <!-- Router Outlet Canvas -->
        <div class="flex-1 relative overflow-hidden flex flex-col">
          <router-outlet></router-outlet>
        </div>
      </main>
    </div>
  `,
})
export class TenantLayoutComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private router = inject(Router);

  isCollapsed = signal(false);
  tenantId = signal('');
  pageTitle = signal('');

  tenantName = computed(() => {
    const id = this.tenantId();
    if (!id) return 'Cargando...';
    // Formatear el tenantId (ej: acme-corp -> Acme Corp)
    return id
      .split('-')
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  });

  tenantMembers = signal(16); // Placeholder como en la imagen

  ngOnInit() {
    this.updateTenantId();
    this.updatePageTitle(this.router.url);

    this.router.events.pipe(filter((event) => event instanceof NavigationEnd)).subscribe((event: any) => {
      this.updateTenantId();
      this.updatePageTitle(event.urlAfterRedirects);
    });
  }

  toggleSidebar() {
    this.isCollapsed.update((v) => !v);
  }

  private updateTenantId() {
    // Traverse the route tree to find the tenantId param since it might be in children or parent
    let currentRoute: ActivatedRoute | null = this.route;
    while (currentRoute) {
      if (currentRoute.snapshot.paramMap.has('tenantId')) {
        this.tenantId.set(currentRoute.snapshot.paramMap.get('tenantId')!);
        break;
      }
      currentRoute = currentRoute.firstChild;
    }
  }

  private updatePageTitle(url: string) {
    if (url.includes('/inbox')) {
      this.pageTitle.set('Inbox');
    } else if (url.includes('/invoices')) {
      this.pageTitle.set('Invoices');
    } else {
      this.pageTitle.set('Dashboard');
    }
  }
}

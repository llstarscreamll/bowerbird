import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { publicGuard } from './core/guards/public.guard';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./health/presentation/pages/home/home.component').then((c) => c.HomeComponent),
  },
  {
    path: 'login',
    canActivate: [publicGuard],
    loadComponent: () => import('./auth/presentation/pages/login/login.component').then((c) => c.LoginComponent),
  },
  {
    path: 'lobby',
    canActivate: [authGuard],
    loadComponent: () => import('./auth/presentation/pages/lobby/lobby.component').then((c) => c.LobbyComponent),
  },
  {
    path: ':tenantId',
    canActivate: [authGuard],
    loadComponent: () => import('./core/presentation/layouts/tenant-layout/tenant-layout.component').then((c) => c.TenantLayoutComponent),
    children: [
      {
        path: 'inbox/connections',
        loadComponent: () => import('./inbox/presentation/pages/connections/inbox-connections.component').then((c) => c.InboxConnectionsComponent),
      },
      {
        path: 'inbox/unified',
        loadComponent: () => import('./inbox/presentation/pages/unified/unified-inbox.component').then((c) => c.UnifiedInboxComponent),
      },
    ],
  },
];

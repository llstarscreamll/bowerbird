import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { publicGuard } from './core/guards/public.guard';
import { platformOperatorGuard } from './core/guards/platform-operator.guard';

export const routes: Routes = [
  {
    path: '',
    redirectTo: 'login',
    pathMatch: 'full',
  },
  {
    path: 'login',
    canActivate: [publicGuard],
    loadComponent: () => import('./auth/presentation/pages/login/login.page').then((c) => c.LoginPage),
  },
  {
    path: 'lobby',
    canActivate: [authGuard],
    loadComponent: () => import('./auth/presentation/pages/lobby/lobby.page').then((c) => c.LobbyPage),
  },
  {
    path: 'platform',
    canActivate: [authGuard, platformOperatorGuard],
    loadComponent: () => import('./entitlements/presentation/pages/platform/platform.page').then((c) => c.PlatformPage),
  },
  {
    path: ':tenantId',
    canActivate: [authGuard],
    loadComponent: () => import('./core/presentation/layouts/tenant-layout/tenant-layout.component').then((c) => c.TenantLayoutComponent),
    children: [
      {
        path: '',
        redirectTo: 'dashboard',
        pathMatch: 'full',
      },
      {
        path: 'dashboard',
        loadComponent: () => import('./dashboard/presentation/pages/dashboard/dashboard.page').then((c) => c.DashboardPage),
      },
      {
        path: 'inbox/master',
        loadComponent: () => import('./inbox/presentation/pages/master/master.page').then((c) => c.MasterPage),
      },
      {
        path: 'connections',
        loadComponent: () => import('./connections/presentation/pages/master/master.page').then((c) => c.MasterPage),
      },
      {
        path: 'connections/:connectionId',
        loadComponent: () => import('./connections/presentation/pages/detail/detail.page').then((c) => c.DetailPage),
      },
      {
        path: 'invoices',
        loadComponent: () => import('./invoices/presentation/pages/master/master.page').then((c) => c.MasterPage),
      },
      {
        path: 'invoices/:invoiceId',
        loadComponent: () => import('./invoices/presentation/pages/detail/detail.page').then((c) => c.DetailPage),
      },
    ],
  },
];

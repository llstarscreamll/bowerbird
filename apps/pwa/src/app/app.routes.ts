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
        path: '',
        redirectTo: 'dashboard',
        pathMatch: 'full',
      },
      {
        path: 'dashboard',
        loadComponent: () => import('./dashboard/presentation/pages/dashboard/dashboard.component').then((c) => c.DashboardComponent),
      },
      {
        path: 'inbox/master',
        loadComponent: () => import('./inbox/presentation/pages/master/master-inbox.component').then((c) => c.MasterInboxComponent),
      },
      {
        path: 'connections',
        loadComponent: () => import('./connections/presentation/pages/connections-list/connections-list.component').then((c) => c.ConnectionsListComponent),
      },
      {
        path: 'connections/:connectionId',
        loadComponent: () => import('./connections/presentation/pages/connection-details/connection-details.component').then((c) => c.ConnectionDetailsComponent),
      },
      {
        path: 'invoices',
        loadComponent: () => import('./invoices/presentation/pages/master/master-invoices.component').then((c) => c.MasterInvoicesComponent),
      },
    ],
  },
];

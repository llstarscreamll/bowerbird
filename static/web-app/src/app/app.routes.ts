import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () =>
      import('./pages/landing.page').then((m) => m.LandingPageComponent),
  },
  {
    path: 'dashboard',
    loadComponent: () =>
      import('./pages/dashboard.page').then((m) => m.DashboardPageComponent),
  },
];

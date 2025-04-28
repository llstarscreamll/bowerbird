import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    pathMatch: 'full',
    loadComponent: () => import('./pages/landing.page').then((m) => m.LandingPageComponent),
  },
  {
    path: 'dashboard',
    loadComponent: () => import('./pages/dashboard.page').then((m) => m.DashboardPageComponent),
  },
  {
    path: 'wallets/:walletID/transactions/:transactionID',
    loadComponent: () => import('./pages/transaction-detail.page').then((m) => m.TransactionDetailPage),
  },
  {
    path: 'wallets/:walletID/categories/create',
    loadComponent: () => import('./pages/create-category.page').then((m) => m.CreateCategoryPage),
  },
];

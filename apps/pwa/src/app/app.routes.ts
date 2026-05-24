import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    loadComponent: () => import('./health/presentation/pages/home/home.component').then((c) => c.HomeComponent),
  },
  {
    path: 'login',
    loadComponent: () => import('./auth/presentation/pages/login/login.component').then((c) => c.LoginComponent),
  },
  {
    path: 'lobby',
    loadComponent: () => import('./auth/presentation/pages/lobby/lobby.component').then((c) => c.LobbyComponent),
  },
];

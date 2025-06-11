import { BarChart, PieChart } from 'echarts/charts';
import { GridComponent, LegendComponent, TooltipComponent } from 'echarts/components';
import * as echarts from 'echarts/core';
import { EChartsCoreOption } from 'echarts/core';
import { CanvasRenderer } from 'echarts/renderers';
import { initFlowbite } from 'flowbite';
import { NgxEchartsDirective, provideEchartsCore } from 'ngx-echarts';

import { Observable, filter, tap } from 'rxjs';

import { Store } from '@ngrx/store';

import { CommonModule } from '@angular/common';
import { Component, OnInit, inject } from '@angular/core';
import { RouterModule } from '@angular/router';

import { getUser } from '@app/ngrx/auth';
import * as finance from '@app/ngrx/finance';
import { WalletMetrics } from '@app/ngrx/finance';
import { FlowbiteService } from '@app/services/flowbite.service';
import { Wallet } from '@app/types';
import { environment } from '@env/environment';

echarts.use([PieChart, GridComponent, CanvasRenderer, TooltipComponent]);

const colors = {
  'bg-red-300': 'rgb(252, 165, 165)',
  'bg-red-400': 'rgb(248, 113, 113)',
  'bg-red-500': 'rgb(239, 68, 68)',
  'bg-red-600': 'rgb(220, 38, 38)',
  'bg-red-700': 'rgb(185, 28, 28)',
  'bg-orange-300': 'rgb(253, 186, 116)',
  'bg-orange-400': 'rgb(251, 146, 60)',
  'bg-orange-500': 'rgb(249, 115, 22)',
  'bg-orange-600': 'rgb(234, 88, 12)',
  'bg-orange-700': 'rgb(194, 65, 12)',
  'bg-amber-300': 'rgb(252, 211, 77)',
  'bg-amber-400': 'rgb(251, 191, 36)',
  'bg-amber-500': 'rgb(245, 158, 11)',
  'bg-amber-600': 'rgb(217, 119, 6)',
  'bg-amber-700': 'rgb(180, 83, 9)',
  'bg-yellow-300': 'rgb(253, 224, 71)',
  'bg-yellow-400': 'rgb(250, 204, 21)',
  'bg-yellow-500': 'rgb(234, 179, 8)',
  'bg-yellow-600': 'rgb(202, 138, 4)',
  'bg-yellow-700': 'rgb(161, 98, 7)',
  'bg-lime-300': 'rgb(190, 242, 100)',
  'bg-lime-400': 'rgb(163, 230, 53)',
  'bg-lime-500': 'rgb(132, 204, 22)',
  'bg-lime-600': 'rgb(101, 163, 13)',
  'bg-lime-700': 'rgb(77, 124, 15)',
  'bg-green-300': 'rgb(134, 239, 172)',
  'bg-green-400': 'rgb(74, 222, 128)',
  'bg-green-500': 'rgb(34, 197, 94)',
  'bg-green-600': 'rgb(22, 163, 74)',
  'bg-green-700': 'rgb(21, 128, 61)',
  'bg-emerald-300': 'rgb(110, 231, 183)',
  'bg-emerald-400': 'rgb(52, 211, 153)',
  'bg-emerald-500': 'rgb(16, 185, 129)',
  'bg-emerald-600': 'rgb(5, 150, 105)',
  'bg-emerald-700': 'rgb(4, 120, 87)',
  'bg-teal-300': 'rgb(94, 234, 212)',
  'bg-teal-400': 'rgb(45, 212, 191)',
  'bg-teal-500': 'rgb(20, 184, 166)',
  'bg-teal-600': 'rgb(13, 148, 136)',
  'bg-teal-700': 'rgb(15, 118, 110)',
  'bg-cyan-300': 'rgb(103, 232, 249)',
  'bg-cyan-400': 'rgb(34, 211, 238)',
  'bg-cyan-500': 'rgb(6, 182, 212)',
  'bg-cyan-600': 'rgb(8, 145, 178)',
  'bg-cyan-700': 'rgb(14, 116, 144)',
  'bg-sky-300': 'rgb(125, 211, 252)',
  'bg-sky-400': 'rgb(56, 189, 248)',
  'bg-sky-500': 'rgb(14, 165, 233)',
  'bg-sky-600': 'rgb(2, 132, 199)',
  'bg-sky-700': 'rgb(3, 105, 161)',
  'bg-blue-300': 'rgb(147, 197, 253)',
  'bg-blue-400': 'rgb(96, 165, 250)',
  'bg-blue-500': 'rgb(59, 130, 246)',
  'bg-blue-600': 'rgb(37, 99, 235)',
  'bg-blue-700': 'rgb(29, 78, 216)',
  'bg-indigo-300': 'rgb(165, 180, 252)',
  'bg-indigo-400': 'rgb(129, 140, 248)',
  'bg-indigo-500': 'rgb(99, 102, 241)',
  'bg-indigo-600': 'rgb(79, 70, 229)',
  'bg-indigo-700': 'rgb(67, 56, 202)',
  'bg-violet-300': 'rgb(196, 181, 253)',
  'bg-violet-400': 'rgb(167, 139, 250)',
  'bg-violet-500': 'rgb(139, 92, 246)',
  'bg-violet-600': 'rgb(124, 58, 237)',
  'bg-violet-700': 'rgb(109, 40, 217)',
  'bg-purple-300': 'rgb(216, 180, 254)',
  'bg-purple-400': 'rgb(192, 132, 252)',
  'bg-purple-500': 'rgb(168, 85, 247)',
  'bg-purple-600': 'rgb(147, 51, 234)',
  'bg-purple-700': 'rgb(126, 34, 206)',
  'bg-fuchsia-300': 'rgb(240, 171, 252)',
  'bg-fuchsia-400': 'rgb(232, 121, 249)',
  'bg-fuchsia-500': 'rgb(217, 70, 239)',
  'bg-fuchsia-600': 'rgb(192, 38, 211)',
  'bg-fuchsia-700': 'rgb(162, 28, 175)',
  'bg-pink-300': 'rgb(249, 168, 212)',
  'bg-pink-400': 'rgb(244, 114, 182)',
  'bg-pink-500': 'rgb(236, 72, 153)',
  'bg-pink-600': 'rgb(219, 39, 119)',
  'bg-pink-700': 'rgb(190, 24, 93)',
  'bg-rose-300': 'rgb(253, 164, 175)',
  'bg-rose-400': 'rgb(251, 113, 133)',
  'bg-rose-500': 'rgb(244, 63, 94)',
  'bg-rose-600': 'rgb(225, 29, 72)',
  'bg-rose-700': 'rgb(190, 18, 60)',
  'bg-slate-300': 'rgb(203, 213, 225)',
  'bg-slate-400': 'rgb(148, 163, 184)',
  'bg-slate-500': 'rgb(100, 116, 139)',
  'bg-slate-600': 'rgb(71, 85, 105)',
  'bg-slate-700': 'rgb(51, 65, 85)',
  'bg-stone-300': 'rgb(214, 211, 209)',
  'bg-stone-400': 'rgb(168, 162, 158)',
  'bg-stone-500': 'rgb(120, 113, 108)',
  'bg-stone-600': 'rgb(87, 83, 78)',
  'bg-stone-700': 'rgb(68, 64, 60)',
};

@Component({
  imports: [CommonModule, RouterModule, NgxEchartsDirective],
  selector: 'app-dashboard-page',
  templateUrl: './dashboard.page.html',
  providers: [provideEchartsCore({ echarts })],
})
export class DashboardPageComponent implements OnInit {
  private store = inject(Store);
  private flowbite = inject(FlowbiteService);

  user$ = this.store.select(getUser);
  metrics$!: Observable<WalletMetrics | null>;
  selectedWallet$!: Observable<Wallet | null>;
  isLoading$ = this.store.select(finance.getIsLoading);
  transactions$ = this.store.select(finance.getTransactions);

  currentDate = new Date();
  selectedWalletID: string = '';
  apiUrl = environment.apiBaseUrl;
  expensesChartOptions: any = {
    tooltip: {
      trigger: 'item',
    },
    series: [
      {
        name: 'Categorías de gastos',
        type: 'pie',
        radius: ['30%', '75%'],
        avoidLabelOverlap: false,
        padAngle: 5,
        itemStyle: {
          borderRadius: 6,
        },
        label: {
          show: true,
          color: 'oklch(0.704 0.04 256.788)',
          fontSize: 10,
        },
        data: [],
      },
    ],
  };

  ngOnInit(): void {
    this.store.dispatch(finance.actions.getWallets());

    this.selectedWallet$ = this.store.select(finance.getSelectedWallet).pipe(
      filter((w): w is Wallet => w !== null),
      tap((w) => (this.selectedWalletID = w.ID)),
      tap(() => this.store.dispatch(finance.actions.getTransactions({ walletID: this.selectedWalletID }))),
      tap(() =>
        this.store.dispatch(
          finance.actions.getMetrics({
            walletID: this.selectedWalletID,
            from: new Date(this.currentDate.getFullYear(), this.currentDate.getMonth(), 1),
            to: new Date(this.currentDate.getFullYear(), this.currentDate.getMonth() + 1, 0),
          }),
        ),
      ),
      tap(() => this.flowbite.load(() => initFlowbite())),
    );

    this.metrics$ = this.store.select(finance.getMetrics).pipe(
      filter((m) => m !== null),
      tap(
        (m) =>
          (this.expensesChartOptions['series'][0]['data'] = m?.expensesByCategory.map((c) => ({
            value: Math.abs(c.total),
            name: c.categoryName,
            itemStyle: { color: colors[c.color as keyof typeof colors] || 'gray' },
          }))),
      ),
    );
  }

  syncWalletTransactionsWithEmails() {
    this.store.dispatch(finance.actions.syncTransactionsFromEmail({ walletID: this.selectedWalletID }));
  }
}

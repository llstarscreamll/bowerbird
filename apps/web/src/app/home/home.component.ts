import { ChangeDetectionStrategy, Component, OnInit, signal } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [],
  template: `
    <main class="mx-auto flex min-h-screen max-w-4xl flex-col items-center justify-center px-6">
      <section class="w-full rounded-2xl bg-white/80 p-8 shadow-xl backdrop-blur-sm">
        <p class="mb-2 text-sm uppercase tracking-[0.2em] text-cyan-600">Turno Platform</p>
        <h1 class="mb-4 text-4xl font-bold">Angular SPA + Go API</h1>
        <p class="mb-6 text-slate-700">Estado del backend:</p>
        <div class="inline-flex items-center gap-2 rounded-full bg-slate-100 px-4 py-2 text-sm">
          <span class="h-2.5 w-2.5 rounded-full" [class.bg-emerald-500]="status() === 'ok'" [class.bg-orange-500]="status() !== 'ok'"></span>
          <span>{{ status() }}</span>
        </div>
      </section>
    </main>
  `,
  changeDetection: ChangeDetectionStrategy.OnPush
})
export class HomeComponent implements OnInit {
  status = signal('checking...');

  constructor(private readonly httpClient: HttpClient) {}

  ngOnInit(): void {
    this.httpClient
      .get<{ status: string }>('/api/health')
      .subscribe({ next: (res) => this.status.set(res.status), error: () => this.status.set('degraded') });
  }
}

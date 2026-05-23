import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting, HttpTestingController } from '@angular/common/http/testing';
import { HomeComponent } from './home.component';
import { environment } from '../../../../../environments/environment';
import { HEALTH_REPOSITORY } from '../../../domain/health.repository';
import { HealthHttpService } from '../../../infrastructure/health.http.service';

describe('HomeComponent', () => {
  let component: HomeComponent;
  let fixture: ComponentFixture<HomeComponent>;
  let httpMock: HttpTestingController;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HomeComponent],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: HEALTH_REPOSITORY, useClass: HealthHttpService },
      ],
    }).compileComponents();

    httpMock = TestBed.inject(HttpTestingController);
    fixture = TestBed.createComponent(HomeComponent);
    component = fixture.componentInstance;

    // Al usar provideZonelessChangeDetection, detectChanges inicia el ciclo de vida y llama ngOnInit (que dispara la request)
    fixture.detectChanges();

    // Ahora mockeamos la request que dispara el HealthStore a traves de HealthHttpService
    const req = httpMock.expectOne(`${environment.apiUrl}/api/health`);
    req.flush({ status: 'ok' });
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

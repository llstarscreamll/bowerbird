import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ConnectionStatusChipComponent } from './connection-status-chip.component';

describe('ConnectionStatusChipComponent', () => {
  let fixture: ComponentFixture<ConnectionStatusChipComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConnectionStatusChipComponent],
    }).compileComponents();

    fixture = TestBed.createComponent(ConnectionStatusChipComponent);
  });

  it('should render active label and update when status changes', () => {
    fixture.componentRef.setInput('status', 'active');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Activa');

    fixture.componentRef.setInput('status', 'error');
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Error');
    expect(fixture.nativeElement.textContent).not.toContain('Activa');
  });
});

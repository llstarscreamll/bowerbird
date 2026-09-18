import { TestBed } from '@angular/core/testing';
import { ThemeService } from './theme.service';

describe('ThemeService', () => {
  it('should track dark mode from document class', () => {
    document.documentElement.classList.add('dark');

    const service = TestBed.configureTestingModule({}).runInInjectionContext(() => new ThemeService());

    expect(service.isDark()).toBe(true);

    document.documentElement.classList.remove('dark');
    document.documentElement.classList.add('dark');
    expect(service.isDark()).toBe(true);
  });
});

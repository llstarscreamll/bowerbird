import { bootstrapApplication } from '@angular/platform-browser';
import { appConfig } from './app/app.config';
import { AppComponent } from './app/app.component';

// --- Modern Web Best Practices (Accessible Error Announcement / Required Field Feedback) ---
const updateAriaState = (event: Event) => {
  const input = event.target as HTMLElement;
  if (!input.matches?.('input, textarea, select')) return;

  const isUserInvalid = input.matches(':user-invalid');
  if (isUserInvalid) {
    input.setAttribute('aria-invalid', 'true');
  } else {
    input.removeAttribute('aria-invalid');
  }
};

document.addEventListener('blur', updateAriaState, true);
document.addEventListener('focus', updateAriaState, true);
document.addEventListener('input', (event: Event) => {
  const input = event.target as HTMLElement;
  if (!input.matches?.('input, textarea, select')) return;

  const hasAriaInvalid = input.hasAttribute('aria-invalid');
  const ariaInvalid = input.getAttribute('aria-invalid');
  if (hasAriaInvalid && ariaInvalid === 'true') {
    updateAriaState(event);
  }
});
// ----------------------------------------------------------------------------------------

bootstrapApplication(AppComponent, appConfig).catch((err) => console.error(err));

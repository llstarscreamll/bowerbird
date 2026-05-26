import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';
import type { LocalUserCredentials } from '../support/user.factory';

export class LoginPage {
  constructor(private readonly page: Page) {}

  async goto(): Promise<void> {
    await this.page.goto('/login');
  }

  async expectReady(): Promise<void> {
    await expect(this.page.getByRole('heading', { name: 'Inicia sesion' })).toBeVisible();
    await expect(this.page.locator('#email')).toBeVisible();
    await expect(this.page.locator('#password')).toBeVisible();
  }

  async loginWithEmailPassword(credentials: LocalUserCredentials): Promise<void> {
    await this.page.locator('#email').fill(credentials.email);
    await this.page.locator('#password').fill(credentials.password);
    await this.page.getByRole('button', { name: 'Iniciar sesion' }).click();
  }

  async expectLoginError(): Promise<void> {
    await expect(this.page.getByText('Login failed')).toBeVisible();
  }

  async expectStillOnLogin(): Promise<void> {
    await expect(this.page).toHaveURL(/\/login$/);
  }
}

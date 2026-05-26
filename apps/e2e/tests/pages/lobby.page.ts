import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';

export class LobbyPage {
  constructor(private readonly page: Page) {}

  async expectAtLobby(): Promise<void> {
    await expect(this.page).toHaveURL(/\/lobby$/);
    await expect(this.page.getByRole('heading', { name: 'Bienvenido' })).toBeVisible();
  }

  async expectOrganizationSection(): Promise<void> {
    await expect(this.page.getByRole('heading', { name: 'Tus Organizaciones' })).toBeVisible();
  }
}

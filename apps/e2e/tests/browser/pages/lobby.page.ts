import type { Page } from '@playwright/test';
import { expect } from '@playwright/test';

export class LobbyPage {
  constructor(private readonly page: Page) {}

  private tenantListItem(tenantName: string) {
    return this.page.getByRole('listitem').filter({ hasText: tenantName });
  }

  async expectAtLobby(): Promise<void> {
    await expect(this.page).toHaveURL(/\/lobby$/);
    await expect(this.page.getByRole('heading', { name: 'Bienvenido' })).toBeVisible();
  }

  async expectOrganizationSection(): Promise<void> {
    await expect(this.page.getByRole('heading', { name: 'Tus Organizaciones' })).toBeVisible();
  }

  async openCreateForm(): Promise<void> {
    await this.page.getByRole('button', { name: 'Crear nueva' }).click();
  }

  async fillCreateForm(name: string, slug?: string): Promise<void> {
    await this.page.getByLabel('Nombre de la Organización').fill(name);
    if (slug) {
      await this.page.getByLabel('URL del espacio / Slug').fill(slug);
    }
  }

  async submitCreateForm(): Promise<void> {
    await this.page.getByRole('button', { name: 'Crear Organización' }).click();
  }

  async expectTenantInList(tenantName: string): Promise<void> {
    await expect(this.tenantListItem(tenantName)).toBeVisible();
  }

  async openTenant(tenantName: string): Promise<void> {
    await this.tenantListItem(tenantName).click();
  }

  async expectHasTenants(): Promise<void> {
    await expect(this.page.getByRole('listitem')).not.toHaveCount(0);
  }
}

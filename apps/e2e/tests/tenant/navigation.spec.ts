import { test } from '../fixtures/test.fixture';
import { expect } from '@playwright/test';

test.describe('Tenant: navegación', () => {
  test('usuario autenticado puede navegar a las diferentes secciones de un tenant', async ({ authApi, loginPage, lobbyPage, newUser, page }) => {
    const orgName = `NavOrg ${Date.now()}`;
    const orgSlug = `navorg-${Date.now()}`;

    await test.step('Dado que un usuario ha iniciado sesión y tiene una organización', async () => {
      await authApi.registerLocalOrFail(newUser);
      await loginPage.goto();
      await loginPage.expectReady();
      await loginPage.loginWithEmailPassword(newUser);
      await lobbyPage.expectAtLobby();

      await lobbyPage.openCreateForm();
      await lobbyPage.fillCreateForm(orgName, orgSlug);
      await lobbyPage.submitCreateForm();
      await lobbyPage.expectTenantInList(orgName);
    });

    await test.step('Cuando hace clic en "Correo" va a la página de conexiones', async () => {
      await page.getByRole('listitem').filter({ hasText: orgName }).getByRole('button', { name: 'Correo' }).click();
      await expect(page).toHaveURL(/.*\/inbox\/connections$/);
      await expect(page.getByRole('heading', { name: 'Conexiones de correo' })).toBeVisible();
      await expect(page.locator('app-alert[type="error"]')).not.toBeVisible();
    });

    await test.step('Y puede navegar a "Bandeja unificada" desde conexiones', async () => {
      await page.getByRole('link', { name: 'Ir a bandeja unificada' }).click();
      await expect(page).toHaveURL(/.*\/inbox\/unified$/);
      await expect(page.getByRole('heading', { name: 'Inbox', exact: true })).toBeVisible();
      await expect(page.locator('app-alert[type="error"]')).not.toBeVisible();
    });

    await test.step('Y puede volver a "Conexiones de correo" desde la bandeja', async () => {
      await page.goBack();
      await expect(page).toHaveURL(/.*\/inbox\/connections$/);
      await expect(page.getByRole('heading', { name: 'Conexiones de correo' })).toBeVisible();
      await expect(page.locator('app-alert[type="error"]')).not.toBeVisible();
    });
  });
});

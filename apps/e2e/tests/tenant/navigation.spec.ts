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

    await test.step('Cuando selecciona la organización va al dashboard del tenant', async () => {
      await lobbyPage.openTenant(orgName);
      await expect(page).toHaveURL(/.*\/dashboard$/);
      await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    });

    await test.step('Y puede navegar a "Mails" desde el menú lateral', async () => {
      await page.getByRole('link', { name: 'Mails' }).click();
      await expect(page).toHaveURL(/.*\/inbox\/master$/);
      await expect(page.getByRole('heading', { level: 2, name: /Inbox/ })).toBeVisible();
    });

    await test.step('Y puede navegar a "Conexiones" desde inbox', async () => {
      await page.getByRole('button', { name: 'Añadir cuenta' }).click();
      await expect(page).toHaveURL(/.*\/connections$/);
      await expect(page.getByRole('heading', { name: 'Conexiones' })).toBeVisible();
    });

    await test.step('Y puede volver a "Mails" desde conexiones', async () => {
      await page.getByRole('link', { name: 'Mails' }).click();
      await expect(page).toHaveURL(/.*\/inbox\/master$/);
      await expect(page.getByRole('heading', { level: 2, name: /Inbox/ })).toBeVisible();
    });
  });
});

import { test } from '../fixtures/test.fixture';
import { expect } from '@playwright/test';

test.describe('Tenant: creación', () => {
  test('usuario puede crear tenant dejando que el slug se autogenere', async ({ authApi, loginPage, lobbyPage, newUser, page }) => {
    const orgName = `Auto Slug ${Date.now()}`;

    await test.step('Dado que un usuario ha iniciado sesión', async () => {
      await authApi.registerLocalOrFail(newUser);
      await loginPage.goto();
      await loginPage.expectReady();
      await loginPage.loginWithEmailPassword(newUser);
      await lobbyPage.expectAtLobby();
    });

    await test.step('Cuando abre el formulario, escribe el nombre y no toca el slug', async () => {
      await lobbyPage.openCreateForm();
      await page.getByLabel('Nombre de la Organización').fill(orgName);
      // Verify slug is auto-generated
      const slugInput = page.getByLabel('URL del espacio / Slug');
      await expect(slugInput).not.toBeEmpty();

      await lobbyPage.submitCreateForm();
    });

    await test.step('Entonces ve la nueva organización en la lista del lobby', async () => {
      await lobbyPage.expectHasTenants();
      await lobbyPage.expectTenantInList(orgName);
    });
  });
});

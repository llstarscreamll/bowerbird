import { test } from '../fixtures/test.fixture';
import { expect } from '@playwright/test';

test.describe('Tenant: Lobby', () => {
  test('usuario puede cerrar sesión desde el lobby', async ({ authApi, loginPage, lobbyPage, newUser, page }) => {
    await test.step('Dado que un usuario ha iniciado sesión', async () => {
      await authApi.registerLocalOrFail(newUser);
      await loginPage.goto();
      await loginPage.expectReady();
      await loginPage.loginWithEmailPassword(newUser);
      await lobbyPage.expectAtLobby();
    });

    await test.step('Cuando hace clic en cerrar sesión', async () => {
      await page.getByRole('button', { name: 'Cerrar sesión' }).click();
    });

    await test.step('Entonces es redirigido al login', async () => {
      await loginPage.expectReady();
    });
  });
});

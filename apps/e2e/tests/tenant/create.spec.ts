import { test } from '../fixtures/test.fixture';

test.describe('Tenant: creación', () => {
  test('usuario autenticado puede crear un nuevo tenant', async ({ authApi, loginPage, lobbyPage, newUser }) => {
    const orgName = `Org ${Date.now()}`;
    const orgSlug = `org-${Date.now()}`;

    await test.step('Dado que un usuario ha iniciado sesión', async () => {
      await authApi.registerLocalOrFail(newUser);
      await loginPage.goto();
      await loginPage.expectReady();
      await loginPage.loginWithEmailPassword(newUser);
      await lobbyPage.expectAtLobby();
    });

    await test.step('Cuando abre el formulario y envía los datos de una nueva organización', async () => {
      await lobbyPage.openCreateForm();
      await lobbyPage.fillCreateForm(orgName, orgSlug);
      await lobbyPage.submitCreateForm();
    });

    await test.step('Entonces ve la nueva organización en la lista del lobby', async () => {
      await lobbyPage.expectHasTenants();
      await lobbyPage.expectTenantInList(orgName);
    });
  });
});

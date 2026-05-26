import { buildLocalUserCredentials } from '../support/user.factory';
import { test } from '../fixtures/test.fixture';

test.describe('Auth: email y password', () => {
  test('visitante crea cuenta y accede al lobby', async ({ authApi, loginPage, lobbyPage, newUser }) => {
    await test.step('Dado que el visitante crea una cuenta local con email y password', async () => {
      await authApi.registerLocalOrFail(newUser);
    });

    await test.step('Cuando inicia sesión desde la pantalla de login', async () => {
      await loginPage.goto();
      await loginPage.expectReady();
      await loginPage.loginWithEmailPassword(newUser);
    });

    await test.step('Entonces llega autenticado al lobby', async () => {
      await lobbyPage.expectAtLobby();
      await lobbyPage.expectOrganizationSection();
    });
  });

  test('usuario recibe error al iniciar sesión con password invalido', async ({ authApi, loginPage }) => {
    const existingUser = buildLocalUserCredentials();

    await test.step('Dado que existe una cuenta local registrada', async () => {
      await authApi.registerLocalOrFail(existingUser);
    });

    await test.step('Cuando intenta iniciar sesión con password incorrecto', async () => {
      await loginPage.goto();
      await loginPage.expectReady();
      await loginPage.loginWithEmailPassword({
        ...existingUser,
        password: `${existingUser.password}-wrong`,
      });
    });

    await test.step('Entonces ve un mensaje de credenciales invalidas y permanece en login', async () => {
      await loginPage.expectLoginError();
      await loginPage.expectStillOnLogin();
    });
  });
});

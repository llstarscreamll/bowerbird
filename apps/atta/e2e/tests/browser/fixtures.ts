import { LobbyPage } from './pages/lobby.page';
import { LoginPage } from './pages/login.page';
import { test as apiTest } from '../support/api.fixture';

type BrowserFixtures = {
  loginPage: LoginPage;
  lobbyPage: LobbyPage;
};

export const test = apiTest.extend<BrowserFixtures>({
  loginPage: async ({ page }, use) => {
    await use(new LoginPage(page));
  },
  lobbyPage: async ({ page }, use) => {
    await use(new LobbyPage(page));
  },
});

export { expect } from '../support/api.fixture';

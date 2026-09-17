import { expect } from '@playwright/test';
import { test } from '../fixtures';
import type { LobbyPage } from '../pages/lobby.page';
import type { LoginPage } from '../pages/login.page';
import type { AuthApiClient } from '../../support/auth-api.client';
import type { AuthSession, PlatformApiClient, TenantContext } from '../../support/platform-api.client';
import type { LocalUserCredentials } from '../../support/user.factory';
import { newUlid } from '../../support/ulid';

const CREATE = 'POST /api/v1/catalog/items';

async function createItem(platformApi: PlatformApiClient, auth: AuthSession, tenant: TenantContext, name: string, code: string): Promise<string> {
  const id = newUlid();
  const response = await platformApi.call('/api/v1/catalog/items', {
    method: 'POST',
    auth,
    tenant,
    data: { data: { type: 'catalog_items', id, attributes: { name, kind: 'goods', internal_code: code } } },
  });
  if (!response.ok()) {
    const body = await response.text();
    throw new Error(`${CREATE} failed: HTTP ${response.status()} body=${body}`);
  }
  return id;
}

async function loginAndOpenCatalog(
  authApi: AuthApiClient,
  loginPage: LoginPage,
  lobbyPage: LobbyPage,
  page: import('@playwright/test').Page,
  orgName: string,
  orgSlug: string,
  newUser: LocalUserCredentials,
): Promise<void> {
  await authApi.registerLocalOrFail(newUser);
  await loginPage.goto();
  await loginPage.expectReady();
  await loginPage.loginWithEmailPassword(newUser);
  await lobbyPage.expectAtLobby();

  if (orgSlug) {
    await lobbyPage.openCreateForm();
    await lobbyPage.fillCreateForm(orgName, orgSlug);
    await lobbyPage.submitCreateForm();
  }

  await lobbyPage.openTenant(orgName);
  await page.getByRole('link', { name: 'Catálogo' }).click();
  await expect(page).toHaveURL(/\/catalog$/);
  await expect(page.getByRole('heading', { level: 1, name: 'Catálogo' })).toBeVisible();
}

test.describe('Catalog: master header', () => {
  test('sin clusters muestra acciones compactas', async ({ authApi, loginPage, lobbyPage, page, newUser }) => {
    const orgName = `CatHdr ${Date.now()}`;
    const orgSlug = `cathdr-${Date.now()}`;

    await loginAndOpenCatalog(authApi, loginPage, lobbyPage, page, orgName, orgSlug, newUser);

    const header = page.locator('header');
    await expect(header.getByRole('link', { name: 'Nuevo' })).toBeVisible();
    await expect(header.getByRole('button', { name: 'Cargar' })).toBeVisible();
    await expect(header.getByRole('link', { name: 'Resolver duplicados' })).toHaveCount(0);
    await expect(header.getByRole('link', { name: 'Cargas masivas' })).toHaveCount(0);
    await expect(page.getByRole('heading', { name: 'Duplicados pendientes' })).toHaveCount(0);

    await header.getByRole('button', { name: 'Cargar' }).click();
    await expect(page.getByRole('heading', { name: 'Cargar catálogo' })).toBeVisible();
    await page.getByRole('button', { name: 'Cancelar' }).click();

    await header.getByRole('button', { name: 'Más acciones de carga' }).click();
    await page.getByRole('menuitem', { name: 'Cargas masivas' }).click();
    await expect(page).toHaveURL(/\/catalog\/imports$/);
  });

  test('con clusters muestra banner de duplicados', async ({ authApi, loginPage, lobbyPage, page, newUser, platformApi }) => {
    const orgName = `CatDup ${Date.now()}`;
    const orgSlug = `catdup-${Date.now()}`;
    const stamp = `${Date.now()}`;
    const name = `Dup Header ${stamp}`;

    await authApi.registerLocalOrFail(newUser);
    const auth = await platformApi.loginLocalOrFail(newUser);
    const tenant = await platformApi.createTenantOrFail(auth, { name: orgName, slug: orgSlug });
    await createItem(platformApi, auth, tenant, name, `INT-HA-${stamp}`);
    await createItem(platformApi, auth, tenant, name, `INT-HB-${stamp}`);

    await loginPage.goto();
    await loginPage.expectReady();
    await loginPage.loginWithEmailPassword(newUser);
    await lobbyPage.expectAtLobby();
    await lobbyPage.openTenant(orgName);
    await page.getByRole('link', { name: 'Catálogo' }).click();
    await expect(page).toHaveURL(/\/catalog$/);

    const header = page.locator('header');
    await expect(header.getByRole('link', { name: 'Resolver duplicados' })).toHaveCount(0);

    const banner = page.getByRole('link').filter({ has: page.getByRole('heading', { name: 'Duplicados pendientes' }) });
    await expect(banner).toBeVisible();
    await banner.click();
    await expect(page).toHaveURL(/\/catalog\/duplicates$/);
  });
});

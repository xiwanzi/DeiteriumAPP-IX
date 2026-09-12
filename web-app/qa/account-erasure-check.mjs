async (page) => {
  const target = page.getByRole('row').filter({ hasText: 'EraseFixture' });
  await target.getByRole('button', { name: '永久注销', exact: true }).click();
  const input = page.getByRole('textbox', { name: '当前管理员密码' });
  const confirm = page.getByRole('button', { name: '确认永久注销' });
  if (!await confirm.isDisabled()) throw new Error('Empty password enabled deletion');
  await page.screenshot({ path: 'docs/qa/artifacts/account-erasure/web-confirmation.png', fullPage: true });
  await input.fill('wrong-password'); await confirm.click();
  await page.getByRole('alert').filter({ hasText: '当前登录的管理员密码不正确' }).waitFor();
  if (await input.inputValue() !== '') throw new Error('Password remained after failure');
  if (await page.evaluate(() => window.erasureQA.mutations) !== 0) throw new Error('Bad password deleted a user');
  await page.evaluate(() => window.erasureQA.mode = 'unknown');
  await input.fill('qa-admin-password'); await confirm.click();
  await page.getByRole('alert').filter({ hasText: '连接中断' }).waitFor();
  if (await input.inputValue() !== '') throw new Error('Password remained after unknown response');
  await page.evaluate(() => window.erasureQA.mode = 'normal');
  await input.fill('qa-admin-password'); await confirm.click();
  await page.getByRole('dialog').waitFor({ state: 'hidden' });
  await page.getByRole('row').filter({ hasText: 'EraseFixture' }).waitFor({ state: 'hidden' });
  const observed = await page.evaluate(() => ({ mutations: window.erasureQA.mutations, requests: window.erasureQA.requests, users: window.erasureQA.rows.map(r => r.gameId) }));
  if (observed.mutations !== 1 || new Set(observed.requests.map(r => r.requestId)).size !== 1) throw new Error('Unknown retry changed request identity or duplicated deletion');
  if (observed.users.join(',') !== 'AdminFixture,KeepFixture') throw new Error('Unrelated account changed');
  await page.screenshot({ path: 'docs/qa/artifacts/account-erasure/web-completed.png', fullPage: true });
  return observed;
}

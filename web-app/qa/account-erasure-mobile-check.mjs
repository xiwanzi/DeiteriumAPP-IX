async (page) => {
  await page.reload();
  await page.setViewportSize({width:390,height:844});
  await page.evaluate(()=>{document.documentElement.dataset.theme='dark';window.erasureQA.mode='delay';});
  await page.getByRole('row').filter({hasText:'EraseFixture'}).getByRole('button',{name:'永久注销',exact:true}).click();
  const input=page.getByRole('textbox',{name:'当前管理员密码'});
  await input.fill('qa-admin-password');
  await page.screenshot({path:'docs/qa/artifacts/account-erasure/web-mobile-dark.png',fullPage:true});
  await page.getByRole('button',{name:'确认永久注销'}).click();
  await page.waitForFunction(()=>typeof window.erasureQA.release==='function');
  if(!await page.getByRole('button',{name:'取消',exact:true}).isDisabled())throw new Error('Cancel enabled during deletion');
  if(!await input.isDisabled())throw new Error('Password editable during deletion');
  await page.getByRole('button',{name:'关闭弹窗'}).click();
  if(!await page.getByRole('dialog').isVisible())throw new Error('Pending dialog was closed');
  const rect=await page.getByRole('dialog').boundingBox();
  if(rect.x<0||rect.x+rect.width>391)throw new Error('Modal overflows mobile width');
  await page.evaluate(()=>window.erasureQA.release());
  await page.getByRole('dialog').waitFor({state:'hidden'});
  return {mobileAndPendingControls:'PASS'};
}
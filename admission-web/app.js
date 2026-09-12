const form = document.querySelector('#entry-form');
const fields = [...document.querySelectorAll('[data-step]')];
const steps = [...document.querySelectorAll('.steps li')];
const id = document.querySelector('#game-id');
const qq = document.querySelector('#qq');
const covenant = document.querySelector('#covenant');
const phase = document.querySelector('.phase-index');
const intro = document.querySelector('#step-intro');
let step = 0;
let submitting = false;
let pendingSubmission = null;
let currentReceipt = '';
let serviceConfig = null;
const COVENANT_VERSION = '2026-09-12-v1';
const PENDING_KEY = 'deuterium-admission-pending-v1';
const RECEIPT_KEY = 'deuterium-admission-receipt-v1';
const storage = {
  read(key, session = false) { try { return (session ? sessionStorage : localStorage).getItem(key); } catch { return null; } },
  write(key, value, session = false) { try { (session ? sessionStorage : localStorage).setItem(key, value); } catch {} },
  remove(key, session = false) { try { (session ? sessionStorage : localStorage).removeItem(key); } catch {} },
};

function showStep(value, focus = true) {
  step = value;
  fields.forEach((field, i) => { field.hidden = i !== step; });
  steps.forEach((item, i) => {
    item.classList.toggle('active', i === step);
    item.classList.toggle('done', i < step);
    if (i === step) item.setAttribute('aria-current', 'step'); else item.removeAttribute('aria-current');
  });
  phase.replaceChildren(document.createTextNode(`0${step + 1}`), Object.assign(document.createElement('small'), {textContent:'/03'}));
  document.querySelector('#form-title').textContent = ['建立身份档案', '选择开拓方向', '核对申请信息'][step];
  document.querySelector('.form-header .eyebrow').textContent = ['IDENTITY REGISTRATION', 'EXPLORATION PREFERENCE', 'PERSONNEL CONFIRMATION'][step];
  intro.textContent = ['填写游戏 ID 和 QQ 号码，申请加入服务器。', '你平时喜欢怎么玩？可以多选，也可以跳过。', '请核对信息。审核结果将发送至你填写的 QQ 邮箱。'][step];
  document.querySelector('#previous').hidden = step === 0;
  document.querySelector('#next-label').textContent = step === 2 ? '提交申请' : step === 1 ? '核对信息' : '下一步';
  document.querySelector('#next-en').textContent = step === 2 ? 'SUBMIT' : 'CONTINUE';
  document.querySelector('#step-counter').textContent = `0${step + 1} / 03`;
  if (step === 2) {
    document.querySelector('#review-id').textContent = id.value.trim();
    document.querySelector('#review-qq').textContent = qq.value.trim();
    document.querySelector('#review-interest').textContent = [...document.querySelectorAll('[name=interest]:checked')].map(x => x.value).join(' / ') || '暂未选择';
    document.querySelector('#review-message').textContent = document.querySelector('#message').value.trim() || '—';
  }
  if (focus) {
    fields[step].querySelector('input,textarea')?.focus({preventScroll:true});
    if (innerWidth <= 850) document.querySelector('.terminal').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth',block:'start'});
  }
}
function validateIdentity() {
  const idValid = /^[A-Za-z0-9_]{3,16}$/.test(id.value.trim());
  const qqValid = /^[1-9][0-9]{4,10}$/.test(qq.value.trim());
  document.querySelector('#id-error').textContent = idValid ? '' : '请输入 3–16 位玩家名：英文字母、数字或下划线。';
  document.querySelector('#qq-error').textContent = qqValid ? '' : '请输入 5–11 位 QQ 号码，首位不能为 0。';
  id.setAttribute('aria-invalid', String(!idValid));
  qq.setAttribute('aria-invalid', String(!qqValid));
  const accepted = covenant.checked;
  document.querySelector('#covenant-error').textContent = accepted ? '' : '请阅读并同意《文明游戏公约》后继续。';
  covenant.setAttribute('aria-invalid', String(!accepted));
  if (!idValid) id.focus(); else if (!qqValid) qq.focus(); else if (!accepted) covenant.focus();
  return idValid && qqValid && accepted;
}
form.addEventListener('submit', async event => {
  event.preventDefault();
  if (submitting) return;
  if (pendingSubmission && step === 2) { await submitApplication(); return; }
  if (step === 0 && !validateIdentity()) return;
  if (step < 2) { showStep(step + 1); return; }
  if (!covenant.checked) { showStep(0); validateIdentity(); return; }
  if (!document.querySelector('#consent').checked) {
    document.querySelector('#consent-error').textContent = '请确认填写的信息真实，并由本人使用。';
    document.querySelector('#consent').focus(); return;
  }
  await submitApplication();
});
document.querySelector('#previous').addEventListener('click', () => showStep(step - 1));
covenant.addEventListener('change', () => {
  document.querySelector('#covenant-error').textContent = '';
  covenant.removeAttribute('aria-invalid');
});
document.querySelector('#message').addEventListener('input', e => { document.querySelector('#word-count').textContent = e.target.value.length; });
document.querySelector('#consent').addEventListener('change', () => { document.querySelector('#consent-error').textContent = ''; });
for (const input of [id, qq]) input.addEventListener('input', () => {
  input.removeAttribute('aria-invalid');
  document.querySelector(input === id ? '#id-error' : '#qq-error').textContent = '';
});
const dialog = document.querySelector('#info-dialog');
const content = {
  guide: {title:'申请前，先看这里', html:'<ul><li>使用你本人持有的 Minecraft Java 版正版账号，填写当前玩家名，不要填写邮箱或 UUID。</li><li><strong>先加入服务器 QQ 群（490579956）。填写的 QQ 号码不在群内，将无法通过审核。</strong></li><li>阅读并同意《文明游戏公约》，确认愿意遵守服务器规则。</li><li>请确认该 QQ 邮箱可以收信，审核结果将发送至“QQ号码@qq.com”。</li></ul><p>申请通过后才能进入服务器。入群或提交申请都不代表已经获得通行许可。</p><a class="dialog-group-link" href="https://qm.qq.com/q/HROB9FDSYE" target="_blank" rel="noopener noreferrer">加入 Deuterium · 柚 QQ 群 ↗</a>'},
  about: {title:'欢迎来到 Deuterium IX', html:'<p>这里是一个 Minecraft 玩家社区。</p><p>你可以盖房子、做自动化、探索新的地方，也可以和朋友一起慢慢建设。无论熟悉模组，还是刚开始接触，都欢迎你来认识我们。</p><p>先加入 QQ 群，看看群公告里的玩法介绍，再申请你的通行许可。遇到问题可以在群内询问管理组。</p>'},
  covenant: {title:'文明游戏公约', html:'<p class="covenant-intro">一起玩得长久，比一时的输赢更重要。加入 Deuterium IX，请和我们一起遵守：</p><ol class="covenant-list"><li><strong>友善交流，尊重彼此</strong><p>不辱骂、骚扰或歧视他人，不公开他人的个人信息。遇到分歧先沟通，不把争执带成围攻。</p></li><li><strong>珍惜他人的建设成果</strong><p>未经允许，不拆改他人建筑、不拿取他人物资。使用他人的设施或进入私人区域前，先征得同意。</p></li><li><strong>公平游玩，不作弊</strong><p>不使用作弊工具，不利用漏洞刷取物资或破坏服务器。发现问题请向管理组反馈。</p></li><li><strong>照顾共同的游戏环境</strong><p>不刷屏、不发送骚扰广告，不恶意制造卡顿或占用服务器资源。建设大型装置时，遵守所在子服的性能规则。</p></li><li><strong>遵守规则，有事沟通</strong><p>阅读群公告和各子服的玩法规则。发生纠纷时保留记录，联系管理组处理，不自行报复或破坏。</p></li></ol><p>各子服的具体玩法与限制，请以群公告为准。</p>'},
  privacy: {title:'资料使用说明', html:'<p>游戏 ID 用于核对账号和设置通行权限；QQ 号码用于核对群成员身份、联系审核事项，并将审核结果发送至对应的“QQ号码@qq.com”邮箱。这些资料不用于公开展示。</p><p>点击“提交申请”后，资料将发送给服务器管理组。请只填写本人信息，不要在补充记录中提供密码、验证码或其他敏感资料。</p><p>查询凭证仅保存在你当前的浏览器中；尚未确认结果的提交内容会暂存在当前标签页，便于重试。申请和处理记录由管理组保存，用于审核与处理争议。需要更正资料或了解审核情况，请在服务器 QQ 群内联系管理组。</p>'},
  source: {title:'影像鸣谢', html:'<p>背景：DSA 火箭发射。</p><p>采用方块风格的夜间发射场景，由 Deuterium IX 提供。</p>'},
};
function openDialog(key) {
  const item = content[key];
  document.querySelector('#dialog-title').textContent = item.title;
  document.querySelector('#dialog-content').innerHTML = item.html;
  document.querySelector('.dialog-ok>span:first-child').textContent = key === 'covenant' ? '我已了解' : '返回申请';
  dialog.showModal();
}
document.querySelectorAll('[data-dialog]').forEach(button => button.addEventListener('click', () => openDialog(button.dataset.dialog)));
document.querySelectorAll('.close-dialog,.dialog-ok').forEach(button => button.addEventListener('click', () => dialog.close()));
dialog.addEventListener('click', event => {
  if (event.target !== dialog) return;
  const r = dialog.getBoundingClientRect();
  if (event.clientX < r.left || event.clientX > r.right || event.clientY < r.top || event.clientY > r.bottom) dialog.close();
});
document.querySelector('#nav-register').addEventListener('click', () => {
  document.querySelector('.terminal').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth'});
  if (!form.hidden) fields[step].querySelector('input,textarea')?.focus({preventScroll:true});
});

async function api(path, payload) {
  let response;
  try { response = await fetch('/api/v1/admission/' + path, { method: payload === undefined ? 'GET' : 'POST', credentials: 'omit', cache: 'no-store', headers: payload === undefined ? { Accept: 'application/json' } : { Accept: 'application/json', 'Content-Type': 'application/json' }, body: payload === undefined ? undefined : JSON.stringify(payload), signal: AbortSignal.timeout(15000) }); }
  catch { throw Object.assign(new Error('暂时无法连接服务器，请稍后重试。'), { status: 0 }); }
  let value;
  try { value = await response.json(); } catch { throw Object.assign(new Error('暂时无法确认服务器的处理结果，请稍后重试。'), { status: 0 }); }
  if (!response.ok || value.error) throw Object.assign(new Error(value.error?.message || '请求未完成，请稍后重试。'), { status: response.status, code: value.error?.code });
  return value.data;
}
async function loadConfig() {
  serviceConfig = await api('config');
  if (serviceConfig.covenantVersion !== COVENANT_VERSION) throw new Error('文明游戏公约已更新，请刷新页面后重新阅读。');
  return serviceConfig;
}
function receiptToken() {
  return btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(32)))).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
}
function lockSubmission(locked) {
  form.querySelectorAll('input,textarea,select').forEach(input => { input.disabled = locked; });
  document.querySelector('#previous').disabled = locked;
  form.querySelectorAll('[data-dialog]').forEach(button => { button.disabled = locked; });
}
function currentPayload() {
  return { receiptToken: receiptToken(), gameId: id.value.trim(), qq: qq.value.trim(), interests: [...form.querySelectorAll('[name=interest]:checked')].map(input => input.value), message: document.querySelector('#message').value.trim(), covenantVersion: COVENANT_VERSION, covenantAccepted: covenant.checked, website: document.querySelector('#website').value };
}
async function submitApplication() {
  if (submitting) return;
  submitting = true;
  const button = document.querySelector('#next'), error = document.querySelector('#submit-error');
  button.disabled = true; error.textContent = ''; document.querySelector('#next-label').textContent = '正在提交…';
  try {
    await loadConfig();
    if (!pendingSubmission) pendingSubmission = currentPayload();
    storage.write(PENDING_KEY, JSON.stringify(pendingSubmission), true);
    currentReceipt = pendingSubmission.receiptToken;
    storage.write(RECEIPT_KEY, currentReceipt);
    lockSubmission(true);
    const result = await api('applications', pendingSubmission);
    pendingSubmission = null; storage.remove(PENDING_KEY, true);
    showReceipt(result, currentReceipt);
  } catch (failure) {
    const uncertain = pendingSubmission && (!failure.status || failure.status >= 500);
    error.textContent = uncertain ? '尚未确认提交结果，资料已保留。请重试提交，或使用“查询申请”确认结果。' : failure.message;
    if (!uncertain) { pendingSubmission = null; storage.remove(PENDING_KEY, true); lockSubmission(false); }
    document.querySelector('#next-label').textContent = uncertain ? '重试提交' : '提交申请';
  } finally { submitting = false; button.disabled = false; }
}
function statusCopy(result) {
  if (result.accessAllowed) return ['通行许可已通过', '你可以使用此正版账号进入服务器了。'];
  if (result.status === 'PENDING') return ['申请已提交，等待审核', '审核结果将发送至你的 QQ 邮箱。请留在服务器群内，也可以随时查询最新结果。'];
  if (result.status === 'REJECTED') return ['申请未通过', '请查看审核原因。有疑问可以在服务器 QQ 群内联系管理组。'];
  if (result.accessStatus === 'REVOKED') return ['通行权限已移除', '如需了解原因，请在服务器 QQ 群内联系管理组。'];
  if (result.status === 'APPROVED') return ['申请已通过', '当前尚无有效通行权限，请联系管理组核实。'];
  return ['申请已提交，等待审核', '审核结果将发送至你的 QQ 邮箱。请留在服务器群内，也可以随时查询最新结果。'];
}
function showReceipt(result, token) {
  currentReceipt = token; storage.write(RECEIPT_KEY, token);
  form.hidden = true; document.querySelector('.steps').hidden = true; document.querySelector('.form-header').hidden = true; intro.hidden = true;
  const output = document.querySelector('#submission-result'); output.hidden = false;
  const [title, detail] = statusCopy(result);
  document.querySelector('#result-title').textContent = title;
  document.querySelector('#result-status').textContent = detail;
  document.querySelector('#result-player').textContent = result.gameId;
  document.querySelector('#result-id').textContent = result.applicationId;
  document.querySelector('#result-date').textContent = new Date(result.createdAt).toLocaleString('zh-CN', { hour12: false });
  document.querySelector('#receipt-token').value = token;
  const reason = document.querySelector('#result-reason'); reason.hidden = result.status !== 'REJECTED' || !result.reason; reason.textContent = result.reason || '';
  document.querySelector('#receipt-feedback').textContent = '';
  output.focus({ preventScroll: true });
  if (innerWidth <= 850) document.querySelector('.terminal').scrollIntoView({ behavior: matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth' });
}
document.querySelector('#copy-receipt').addEventListener('click', async () => {
  try { await navigator.clipboard.writeText(currentReceipt); document.querySelector('#receipt-feedback').textContent = '查询凭证已复制，请妥善保存。'; }
  catch { document.querySelector('#receipt-token').select(); document.querySelector('#receipt-feedback').textContent = '请复制已选中的查询凭证。'; }
});
document.querySelector('#refresh-result').addEventListener('click', async event => {
  const button = event.currentTarget; button.disabled = true;
  try { showReceipt(await api('status', { receiptToken: currentReceipt }), currentReceipt); document.querySelector('#receipt-feedback').textContent = '已更新审核结果。'; }
  catch (failure) { document.querySelector('#receipt-feedback').textContent = failure.message; }
  finally { button.disabled = false; }
});
document.querySelector('#new-application').addEventListener('click', () => {
  if (submitting) return;
  currentReceipt = ''; pendingSubmission = null; storage.remove(RECEIPT_KEY); storage.remove(PENDING_KEY, true);
  form.reset(); lockSubmission(false); document.querySelector('#submit-error').textContent = ''; document.querySelector('#word-count').textContent = '0';
  document.querySelector('#submission-result').hidden = true; form.hidden = false; document.querySelector('.steps').hidden = false; document.querySelector('.form-header').hidden = false; intro.hidden = false; showStep(0);
});
document.querySelector('#query-application').addEventListener('click', () => {
  document.querySelector('#dialog-title').textContent = '查询通行申请';
  document.querySelector('#dialog-content').innerHTML = '<p>输入提交后获得的查询凭证，查看最新审核结果。</p><form id="query-form"><label class="receipt-label" for="query-token">查询凭证</label><input id="query-token" required minlength="43" maxlength="43" autocomplete="off" spellcheck="false"><p class="error" id="query-error" role="alert"></p><button type="submit" class="primary"><span>查询申请</span><span>CHECK STATUS</span></button></form>';
  document.querySelector('#query-token').value = currentReceipt || storage.read(RECEIPT_KEY) || '';
  document.querySelector('.dialog-ok>span:first-child').textContent = '返回申请';
  document.querySelector('#query-form').addEventListener('submit', async event => {
    event.preventDefault(); const token = document.querySelector('#query-token').value.trim(), button = event.target.querySelector('button'); button.disabled = true;
    try { const result = await api('status', { receiptToken: token }); if (pendingSubmission?.receiptToken === token) { pendingSubmission = null; storage.remove(PENDING_KEY, true); } dialog.close(); showReceipt(result, token); }
    catch (failure) { document.querySelector('#query-error').textContent = failure.status === 404 ? '未找到该凭证对应的申请。请核对凭证；刚提交的申请也可稍后再查。' : failure.message; }
    finally { button.disabled = false; }
  });
  dialog.showModal(); document.querySelector('#query-token').focus();
});

(async function restoreSubmission() {
  try {
    const saved = storage.read(PENDING_KEY, true);
    const pending = saved ? JSON.parse(saved) : null;
    if (pending && /^[A-Za-z0-9_-]{43}$/.test(pending.receiptToken) && pending.covenantVersion === COVENANT_VERSION) {
      pendingSubmission = pending; currentReceipt = pending.receiptToken;
      id.value = pending.gameId || ''; qq.value = pending.qq || ''; document.querySelector('#message').value = pending.message || ''; covenant.checked = !!pending.covenantAccepted;
      form.querySelectorAll('[name=interest]').forEach(input => { input.checked = pending.interests?.includes(input.value) || false; }); document.querySelector('#consent').checked = true;
      showStep(2, false); lockSubmission(true); document.querySelector('#next-label').textContent = '重试提交'; document.querySelector('#submit-error').textContent = '上次提交的结果尚未确认，资料已保留。你可以重试提交或查询申请。';
    } else currentReceipt = storage.read(RECEIPT_KEY) || '';
    await loadConfig();
    if (/^[A-Za-z0-9_-]{43}$/.test(currentReceipt)) {
      try { const result = await api('status', { receiptToken: currentReceipt }); pendingSubmission = null; storage.remove(PENDING_KEY, true); showReceipt(result, currentReceipt); }
      catch { /* Keep the pending request for an identical retry. A missing result is not proof of failure. */ }
    }
  } catch { /* Submission reports service and storage errors without blocking the form. */ }
})();

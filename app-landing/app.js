const screenDetails = {
  store: { src: 'assets/provided-store.png', alt: '商城首页 · 用户提供带壳截图', caption: '商城好物，清楚呈现。' },
  chat: { src: 'assets/provided-chat.png', alt: '与 Mori 私聊 · 用户提供带壳截图', caption: '熟悉的朋友，随时相连。' },
  wallet: { src: 'assets/provided-wallet.png', alt: '我的钱包 · 用户提供带壳截图', caption: '每一份收支，都心中有数。' },
  player: { src: 'assets/provided-player.png', alt: 'Mori 的玩家资料 · 用户提供带壳截图', caption: '你的个性，也值得被认识。' },
};
const heroScreen = document.querySelector('#hero-screen');
const screenCaption = document.querySelector('#screen-caption');
let screenTimer;
document.querySelectorAll('[data-screen]').forEach(button => {
  button.addEventListener('click', () => {
    const name = button.dataset.screen;
    document.querySelectorAll('[data-screen]').forEach(item => item.setAttribute('aria-pressed', String(item === button)));
    clearTimeout(screenTimer);
    heroScreen.classList.add('switching');
    screenTimer = setTimeout(() => {
      heroScreen.src = screenDetails[name].src;
      heroScreen.alt = screenDetails[name].alt;
      screenCaption.textContent = screenDetails[name].caption;
      heroScreen.classList.remove('switching');
    }, window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 0 : 140);
  });
});

const webToggle = document.querySelector('#web-toggle');
const webOptions = document.querySelector('#web-options');
function closeOptions() {
  webOptions.hidden = true;
  webToggle.setAttribute('aria-expanded', 'false');
}
webToggle.addEventListener('click', () => {
  const isOpen = webToggle.getAttribute('aria-expanded') === 'true';
  webOptions.hidden = isOpen;
  webToggle.setAttribute('aria-expanded', String(!isOpen));
});
webToggle.addEventListener('keydown', event => {
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    webOptions.hidden = false;
    webToggle.setAttribute('aria-expanded', 'true');
    webOptions.querySelector('button').focus();
  }
});
document.addEventListener('click', event => {
  if (!event.target.closest('.web-split')) closeOptions();
});
document.addEventListener('keydown', event => {
  if (event.key === 'Escape' && !webOptions.hidden) {
    closeOptions();
    webToggle.focus();
  }
});
document.addEventListener('focusin', event => {
  if (!event.target.closest('.web-split')) closeOptions();
});

const dialogReturnTargets = new WeakMap();
function showDialog(dialog) {
  dialogReturnTargets.set(dialog, document.activeElement);
  closeOptions();
  dialog.showModal();
  document.body.style.overflow = 'hidden';
}
const comingDialog = document.querySelector('#coming-dialog');
document.querySelectorAll('[data-coming]').forEach(button => {
  button.addEventListener('click', () => {
    const platform = button.dataset.coming;
    document.querySelector('#coming-platform').textContent = platform;
    document.querySelector('#coming-description').textContent = platform === 'Android App'
      ? 'App 下载暂未开放，敬请期待。'
      : '此入口暂未开放，敬请期待。';
    showDialog(comingDialog);
  });
});
document.querySelectorAll('dialog').forEach(dialog => {
  dialog.querySelectorAll('[data-close]').forEach(button => button.addEventListener('click', () => dialog.close()));
  dialog.addEventListener('click', event => {
    if (event.target !== dialog) return;
    const rect = dialog.getBoundingClientRect();
    if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) dialog.close();
  });
  dialog.addEventListener('close', () => {
    document.body.style.overflow = '';
    const target = dialogReturnTargets.get(dialog);
    if (target?.isConnected && target.getClientRects().length) target.focus();
    else if (target && webOptions.contains(target)) webToggle.focus();
  });
});

if ('IntersectionObserver' in window && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
  document.documentElement.classList.add('js-reveal');
  const observer = new IntersectionObserver(entries => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        entry.target.classList.add('visible');
        observer.unobserve(entry.target);
      }
    });
  }, { threshold: 0.08 });
  document.querySelectorAll('.reveal').forEach(element => observer.observe(element));
}

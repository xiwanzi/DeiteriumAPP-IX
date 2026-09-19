const richMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
const richMobile = window.matchMedia('(max-width: 760px)');
const richHeader = document.querySelector('.site-header');
const richHero = document.querySelector('.hero');
const depthLayers = [...document.querySelectorAll('[data-depth]')];
const journey = document.querySelector('#journey');
const journeySteps = [...document.querySelectorAll('[data-step]')];
const journeyButtons = [...document.querySelectorAll('[data-journey]')];
const journeyScreen = document.querySelector('#journey-screen');
const journeyImages = [
  { src: 'assets/provided-commission.png', alt: '玩家委托详情 · 用户提供带壳截图' },
  { src: 'assets/provided-acceptance.png', alt: '工程已完成、待验收 · 用户提供带壳截图' },
  { src: 'assets/provided-chat.png', alt: '与 Mori 私聊 · 用户提供带壳截图' },
  { src: 'assets/provided-wallet.png', alt: '我的钱包与最近流水 · 用户提供带壳截图' },
];
let journeyIndex = 0;
let journeyChange;
function setJourney(index) {
  if (index === journeyIndex) return;
  journeyIndex = index;
  journeyButtons.forEach((button, i) => button.setAttribute('aria-pressed', String(i === index)));
  journeySteps.forEach((step, i) => step.classList.toggle('is-active', i === index));
  const phone = journeyScreen.parentElement;
  phone.classList.add('is-changing');
  clearTimeout(journeyChange);
  journeyChange = setTimeout(() => {
    journeyScreen.src = journeyImages[index].src;
    journeyScreen.alt = journeyImages[index].alt;
    phone.classList.remove('is-changing');
  }, richMotion.matches ? 0 : 130);
}
journeyButtons.forEach(button => button.addEventListener('click', () => {
  const index = Number(button.dataset.journey);
  setJourney(index);
  if (!richMobile.matches) journeySteps[index].scrollIntoView({ block: 'center', behavior: richMotion.matches ? 'instant' : 'smooth' });
}));

let richFrame = 0;
function updateRichScroll() {
  richFrame = 0;
  const heroBottom = richHero.getBoundingClientRect().bottom;
  richHeader.classList.toggle('is-compact', heroBottom < 95);
  const layerPositions = !richMotion.matches ? depthLayers.map(layer => ({ layer, rect: layer.getBoundingClientRect() })) : [];
  for (const { layer, rect } of layerPositions) {
    if (rect.bottom < -80 || rect.top > innerHeight + 80) continue;
    const progress = (innerHeight / 2 - (rect.top + rect.height / 2)) / innerHeight;
    layer.style.setProperty('--depth-y', `${Math.max(-30, Math.min(30, progress * 28)).toFixed(1)}px`);
  }
  if (!richMobile.matches) {
    const rect = journey.getBoundingClientRect();
    if (rect.top < innerHeight * .78 && rect.bottom > innerHeight * .2) {
      let nearest = 0;
      let distance = Infinity;
      journeySteps.forEach((step, i) => {
        const box = step.getBoundingClientRect();
        const delta = Math.abs(box.top + box.height / 2 - innerHeight * .5);
        if (delta < distance) { nearest = i; distance = delta; }
      });
      setJourney(nearest);
    }
  }
}
function requestRichFrame() { if (!richFrame) richFrame = requestAnimationFrame(updateRichScroll); }
window.addEventListener('scroll', requestRichFrame, { passive: true });
window.addEventListener('resize', requestRichFrame);
window.addEventListener('load', requestRichFrame);
richMotion.addEventListener('change', () => {
  if (richMotion.matches) depthLayers.forEach(layer => layer.style.removeProperty('--depth-y'));
  requestRichFrame();
});
requestRichFrame();

const mediaDialog = document.querySelector('#media-dialog');
function showRichMedia(src, title) {
  document.querySelector('#media-image').src = src;
  document.querySelector('#media-image').alt = title;
  document.querySelector('#media-title').textContent = title;
  showDialog(mediaDialog);
}
document.querySelectorAll('[data-media]').forEach(button => button.addEventListener('click', () => showRichMedia(button.dataset.media, button.dataset.mediaTitle)));

document.querySelectorAll('[data-category]').forEach(button => button.addEventListener('click', () => {
  const category = button.dataset.category;
  document.querySelectorAll('[data-category]').forEach(item => item.setAttribute('aria-pressed', String(item === button)));
  let visible = 0;
  document.querySelectorAll('[data-product-category]').forEach(product => {
    product.hidden = category !== 'all' && product.dataset.productCategory !== category;
    if (!product.hidden) visible++;
  });
  document.querySelector('.demo-products').dataset.visibleCount = String(visible);
  document.querySelector('#category-status').textContent = `当前显示 ${visible} 种 Minecraft 物品`;
}));

const paymentDialog = document.querySelector('#payment-dialog');
const paymentVideo = document.querySelector('#payment-video');
const paymentSound = document.querySelector('#payment-sound');
const soundButton = document.querySelector('#preview-payment-sound');
const paymentNote = document.querySelector('#payment-video-note');
function playPaymentVideo() {
  paymentVideo.currentTime = 0;
  paymentNote.textContent = '体验版 0.9.0 · 原录屏无设备音轨，提示音可单独试听。';
  paymentVideo.play().catch(() => { paymentNote.textContent = '请点击播放器中的播放按钮。原录屏无设备音轨。'; });
}
document.querySelector('#transfer-demo').addEventListener('click', () => {
  showDialog(paymentDialog);
  playPaymentVideo();
});
document.querySelector('#replay-payment').addEventListener('click', playPaymentVideo);
function resetSoundButton() {
  soundButton.disabled = false;
  soundButton.textContent = '试听同版提示音 ♪';
}
soundButton.addEventListener('click', () => {
  paymentSound.currentTime = 0;
  soundButton.disabled = true;
  soundButton.textContent = '正在播放…';
  paymentSound.play().catch(() => { resetSoundButton(); paymentNote.textContent = '提示音暂时未能播放，请重试。'; });
});
paymentSound.addEventListener('ended', resetSoundButton);
paymentDialog.addEventListener('close', () => {
  paymentVideo.pause();
  paymentSound.pause();
  paymentSound.currentTime = 0;
  resetSoundButton();
});

const glassRange = document.querySelector('#glass-range');
glassRange.addEventListener('input', () => {
  document.querySelector('#glass-example').style.setProperty('--glass-blur', `${glassRange.value}px`);
  document.querySelector('#glass-output').textContent = glassRange.value;
});

const galleryItems = [
  { src: 'assets/friends-group.png', title: '热闹，是大家一起创造的。', alt: '玩家们在建筑前合影', bars: false },
  { src: 'assets/street-daylight.png', title: '在街角，遇见喜欢。', alt: '玩家坐在花街的桥边', bars: true },
  { src: 'assets/street-evening.png', title: '等街灯亮起，再慢慢回家。', alt: '暖光照亮的游戏街景', bars: true },
  { src: 'assets/plaza-sunset.png', title: '下一次相聚，还在这里。', alt: '玩家们在夕阳下的樱花广场集合', bars: false },
];
let galleryIndex = 0;
let galleryChange;
function setGallery(index) {
  galleryIndex = (index + galleryItems.length) % galleryItems.length;
  const item = galleryItems[galleryIndex];
  const container = document.querySelector('.gallery-photo');
  container.classList.add('is-changing');
  clearTimeout(galleryChange);
  galleryChange = setTimeout(() => {
    const image = document.querySelector('#gallery-image');
    image.src = item.src;
    image.alt = item.alt;
    container.classList.toggle('has-bars', item.bars);
    document.querySelector('#gallery-caption').textContent = item.title;
    document.querySelector('#gallery-counter').textContent = `${String(galleryIndex + 1).padStart(2, '0')} / 04`;
    container.classList.remove('is-changing');
  }, richMotion.matches ? 0 : 150);
}
document.querySelector('#gallery-prev').addEventListener('click', () => setGallery(galleryIndex - 1));
document.querySelector('#gallery-next').addEventListener('click', () => setGallery(galleryIndex + 1));
document.querySelector('[data-gallery-expand]').addEventListener('click', () => {
  const item = galleryItems[galleryIndex];
  showRichMedia(item.src, item.title);
});
document.querySelector('.world-gallery').addEventListener('keydown', event => {
  if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
    event.preventDefault();
    setGallery(galleryIndex + (event.key === 'ArrowLeft' ? -1 : 1));
  }
});
